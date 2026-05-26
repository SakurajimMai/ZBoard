//go:build ignore

package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"net/mail"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type columnMeta struct {
	Name       string
	DataType   string
	ColumnType string
}

type sourceUser struct {
	SourceID     string
	Email        string
	PasswordHash string
	Balance      string
	PlanID       string
	DeviceLimit  int
	ExpiredAt    *time.Time
	TrafficLimit int64
	TrafficUsed  int64
	Upload       int64
	Download     int64
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type nodeRow struct {
	ID       int64
	Protocol string
}

type report struct {
	SourceTable       string
	SourceTotal       int
	WouldInsert       int
	Inserted          int
	SkippedExisting   int
	SkippedBadEmail   int
	SkippedNoPassword int
	SkippedErrors     int
	ActiveNodes       int
	AutoPlanMatches   int
	WithTargetPlan    int
	WithoutTargetPlan int
	ActiveUsers       int
	DisabledUsers     int
	Provisionable     int
	ExpiredUsers      int
	NoExpiryUsers     int
}

func main() {
	var (
		sourceDSN          = flag.String("source", env("XBOARD_DSN"), "source Xboard MySQL DSN, or XBOARD_DSN")
		targetDSN          = flag.String("target", env("ZBOARD_DSN"), "target Zboard MySQL DSN, or ZBOARD_DSN")
		execute            = flag.Bool("execute", false, "write target database; default is dry-run")
		limit              = flag.Int("limit", 0, "limit source users for testing")
		inspect            = flag.Bool("inspect", false, "print source/target table summaries and exit")
		timeout            = flag.Duration("timeout", 15*time.Minute, "overall migration timeout")
		planMapRaw         = flag.String("plan-map", "", "source:target plan id pairs, e.g. 1:2,3:4")
		defaultPlanID      = flag.Int64("default-plan-id", 0, "target plan id used when no source plan mapping exists")
		autoPlanMapEnabled = flag.Bool("auto-plan-map", true, "map source plans to target plans by exact plan name")
		provisionNodes     = flag.Bool("provision-nodes", true, "create node_users rows for active target nodes")
		copyBalance        = flag.Bool("copy-balance", false, "copy source balance into Zboard")
		balanceCents       = flag.Bool("balance-cents", true, "when copying integer balance, treat it as cents")
	)
	flag.Parse()

	if strings.TrimSpace(*sourceDSN) == "" || strings.TrimSpace(*targetDSN) == "" {
		log.Fatal("missing DSN: set XBOARD_DSN and ZBOARD_DSN, or pass -source/-target")
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	source, err := sql.Open("mysql", *sourceDSN)
	must(err)
	defer source.Close()
	target, err := sql.Open("mysql", *targetDSN)
	must(err)
	defer target.Close()

	must(source.PingContext(ctx))
	must(target.PingContext(ctx))

	sourceUserTable, err := detectTable(ctx, source, []string{"v2_user", "users", "user"})
	must(err)
	sourceCols, err := loadColumns(ctx, source, sourceUserTable)
	must(err)
	targetUserCols, err := loadColumns(ctx, target, "users")
	must(err)
	must(validateTargetUserColumns(targetUserCols))
	must(validateTargetTables(ctx, target))
	if *inspect {
		must(printInspection(ctx, source, target, sourceUserTable, sourceCols))
		return
	}

	explicitPlanMap, err := parsePlanMap(*planMapRaw)
	must(err)
	autoPlanMap := map[string]int64{}
	if *autoPlanMapEnabled {
		autoPlanMap, err = buildAutoPlanMap(ctx, source, target)
		must(err)
	}
	if *defaultPlanID > 0 {
		must(ensureTargetPlanExists(ctx, target, *defaultPlanID))
	}
	for _, id := range explicitPlanMap {
		must(ensureTargetPlanExists(ctx, target, id))
	}

	existingEmails, err := loadExistingEmails(ctx, target)
	must(err)
	targetPlanDeviceLimits, err := loadTargetPlanDeviceLimits(ctx, target)
	must(err)
	activeNodes, err := loadActiveNodes(ctx, target)
	must(err)

	users, err := loadSourceUsers(ctx, source, sourceUserTable, sourceCols, *limit, *copyBalance, *balanceCents)
	must(err)

	rep := report{
		SourceTable:     sourceUserTable,
		SourceTotal:     len(users),
		ActiveNodes:     len(activeNodes),
		AutoPlanMatches: len(autoPlanMap),
	}

	var tx *sql.Tx
	if *execute {
		tx, err = target.BeginTx(ctx, nil)
		must(err)
		defer tx.Rollback()
	}

	for _, u := range users {
		if _, err := mail.ParseAddress(u.Email); err != nil {
			rep.SkippedBadEmail++
			continue
		}
		if strings.TrimSpace(u.PasswordHash) == "" {
			rep.SkippedNoPassword++
			continue
		}
		key := strings.ToLower(strings.TrimSpace(u.Email))
		if existingEmails[key] {
			rep.SkippedExisting++
			continue
		}
		targetPlanID := resolveTargetPlanID(u.PlanID, explicitPlanMap, autoPlanMap, *defaultPlanID)
		rep.WouldInsert++
		if targetPlanID == nil {
			rep.WithoutTargetPlan++
		} else {
			rep.WithTargetPlan++
		}
		if u.Status == "active" {
			rep.ActiveUsers++
		} else {
			rep.DisabledUsers++
		}
		if provisionableSourceUser(u, time.Now().UTC()) {
			rep.Provisionable++
		}
		if u.ExpiredAt == nil {
			rep.NoExpiryUsers++
		} else if !u.ExpiredAt.After(time.Now().UTC()) {
			rep.ExpiredUsers++
		}
		if !*execute {
			continue
		}
		userID, err := insertTargetUser(ctx, tx, u, targetPlanID)
		if err != nil {
			rep.SkippedErrors++
			log.Printf("insert user failed source_id=%s email=%s: %v", u.SourceID, redactEmail(u.Email), err)
			continue
		}
		token, err := randomHex(24)
		if err != nil {
			returnWithRollback(tx, err)
		}
		if err := insertSubToken(ctx, tx, userID, token, u.CreatedAt); err != nil {
			returnWithRollback(tx, err)
		}
		if err := upsertTrafficSnapshot(ctx, tx, userID, u); err != nil {
			returnWithRollback(tx, err)
		}
		if *provisionNodes && provisionableSourceUser(u, time.Now().UTC()) {
			clientID, err := newClientID()
			if err != nil {
				returnWithRollback(tx, err)
			}
			deviceLimit := u.DeviceLimit
			if deviceLimit <= 0 && targetPlanID != nil {
				deviceLimit = targetPlanDeviceLimits[*targetPlanID]
			}
			for _, n := range activeNodes {
				if err := insertNodeUser(ctx, tx, userID, n, clientID, deviceLimit, u.CreatedAt); err != nil {
					returnWithRollback(tx, err)
				}
			}
		}
		existingEmails[key] = true
		rep.Inserted++
	}

	if *execute {
		must(tx.Commit())
	}

	printReport(rep, *execute)
}

func env(name string) string {
	return strings.TrimSpace(strings.Trim(os.Getenv(name), `"`))
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func returnWithRollback(tx *sql.Tx, err error) {
	_ = tx.Rollback()
	log.Fatal(err)
}

func detectTable(ctx context.Context, db *sql.DB, candidates []string) (string, error) {
	for _, name := range candidates {
		var n int
		err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, name).Scan(&n)
		if err != nil {
			return "", err
		}
		if n > 0 {
			return name, nil
		}
	}
	return "", fmt.Errorf("none of candidate tables exists: %s", strings.Join(candidates, ", "))
}

func loadColumns(ctx context.Context, db *sql.DB, table string) (map[string]columnMeta, error) {
	rows, err := db.QueryContext(ctx, `SELECT column_name, data_type, column_type FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ?`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]columnMeta{}
	for rows.Next() {
		var c columnMeta
		if err := rows.Scan(&c.Name, &c.DataType, &c.ColumnType); err != nil {
			return nil, err
		}
		out[strings.ToLower(c.Name)] = c
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("table %s has no columns or does not exist", table)
	}
	return out, rows.Err()
}

func validateTargetUserColumns(cols map[string]columnMeta) error {
	required := []string{"email", "password_hash", "balance", "plan_id", "plan_period", "expired_at", "traffic_limit", "traffic_used", "status"}
	for _, name := range required {
		if _, ok := cols[name]; !ok {
			return fmt.Errorf("target users table missing required column %s", name)
		}
	}
	return nil
}

func validateTargetTables(ctx context.Context, db *sql.DB) error {
	required := []string{"subscription_tokens", "user_traffic_snapshots", "node_users", "plans", "nodes"}
	for _, table := range required {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, table).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("target table %s does not exist; run Zboard migrations first", table)
		}
	}
	return nil
}

func printInspection(ctx context.Context, source, target *sql.DB, sourceUserTable string, sourceUserCols map[string]columnMeta) error {
	var sourceUsers int64
	if err := source.QueryRowContext(ctx, "SELECT COUNT(*) FROM `"+sourceUserTable+"`").Scan(&sourceUsers); err != nil {
		return err
	}
	var targetUsers int64
	if err := target.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&targetUsers); err != nil {
		return err
	}
	var targetNodes int64
	if err := target.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes`).Scan(&targetNodes); err != nil {
		return err
	}
	var activeTargetNodes int64
	if err := target.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes WHERE status = 'active'`).Scan(&activeTargetNodes); err != nil {
		return err
	}

	fmt.Println("source_user_table:", sourceUserTable)
	fmt.Println("source_user_count:", sourceUsers)
	fmt.Println("target_user_count:", targetUsers)
	if err := printTargetPostMigrationStats(ctx, target); err != nil {
		return err
	}
	fmt.Println("target_nodes:", targetNodes)
	fmt.Println("target_active_nodes:", activeTargetNodes)

	if _, ok := sourceUserCols["plan_id"]; ok {
		rows, err := source.QueryContext(ctx, "SELECT COALESCE(CAST(`plan_id` AS CHAR), ''), COUNT(*) FROM `"+sourceUserTable+"` GROUP BY `plan_id` ORDER BY COUNT(*) DESC, `plan_id` ASC")
		if err != nil {
			return err
		}
		defer rows.Close()
		fmt.Println("source_user_plan_distribution:")
		for rows.Next() {
			var planID string
			var count int64
			if err := rows.Scan(&planID, &count); err != nil {
				return err
			}
			if planID == "" {
				planID = "(null)"
			}
			fmt.Printf("  %s: %d\n", planID, count)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if err := printSourcePlanExpiryStats(ctx, source, sourceUserTable, sourceUserCols); err != nil {
			return err
		}
	}
	if _, ok := sourceUserCols["transfer_enable"]; ok {
		if err := printSourceTrafficStats(ctx, source, sourceUserTable, sourceUserCols); err != nil {
			return err
		}
	}
	if err := printSourcePasswordStats(ctx, source, sourceUserTable, sourceUserCols); err != nil {
		return err
	}

	sourcePlanTable, err := detectTable(ctx, source, []string{"v2_plan", "plans", "plan"})
	if err == nil {
		fmt.Println("source_plans:")
		if err := printPlanRows(ctx, source, sourcePlanTable, "  "); err != nil {
			return err
		}
	}
	fmt.Println("target_plans:")
	if err := printPlanRows(ctx, target, "plans", "  "); err != nil {
		return err
	}
	fmt.Println("target_nodes_detail:")
	rows, err := target.QueryContext(ctx, `SELECT id, name, protocol, status FROM nodes ORDER BY id ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name, protocol, status string
		if err := rows.Scan(&id, &name, &protocol, &status); err != nil {
			return err
		}
		fmt.Printf("  %d | %s | %s | %s\n", id, name, protocol, status)
	}
	return rows.Err()
}

func printTargetPostMigrationStats(ctx context.Context, db *sql.DB) error {
	queries := []struct {
		Label string
		Query string
	}{
		{"target_active_subscription_tokens", `SELECT COUNT(*) FROM subscription_tokens WHERE status = 'active'`},
		{"target_traffic_snapshots", `SELECT COUNT(*) FROM user_traffic_snapshots`},
		{"target_node_users", `SELECT COUNT(*) FROM node_users`},
		{"target_users_with_plan", `SELECT COUNT(*) FROM users WHERE plan_id IS NOT NULL`},
		{"target_users_without_plan", `SELECT COUNT(*) FROM users WHERE plan_id IS NULL`},
		{"target_expired_users", `SELECT COUNT(*) FROM users WHERE expired_at IS NOT NULL AND expired_at <= UTC_TIMESTAMP()`},
		{"target_no_expiry_users", `SELECT COUNT(*) FROM users WHERE expired_at IS NULL`},
	}
	for _, item := range queries {
		var n int64
		if err := db.QueryRowContext(ctx, item.Query).Scan(&n); err != nil {
			return err
		}
		fmt.Println(item.Label+":", n)
	}
	rows, err := db.QueryContext(ctx, `SELECT COALESCE(CAST(plan_id AS CHAR), ''), COUNT(*) FROM users GROUP BY plan_id ORDER BY COUNT(*) DESC, plan_id ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()
	fmt.Println("target_user_plan_distribution:")
	for rows.Next() {
		var planID string
		var count int64
		if err := rows.Scan(&planID, &count); err != nil {
			return err
		}
		if planID == "" {
			planID = "(null)"
		}
		fmt.Printf("  %s: %d\n", planID, count)
	}
	return rows.Err()
}

func printSourceTrafficStats(ctx context.Context, db *sql.DB, table string, cols map[string]columnMeta) error {
	fields := []string{"MIN(`transfer_enable`)", "MAX(`transfer_enable`)", "AVG(`transfer_enable`)"}
	if _, ok := cols["u"]; ok {
		fields = append(fields, "MIN(`u`)", "MAX(`u`)")
	}
	if _, ok := cols["d"]; ok {
		fields = append(fields, "MIN(`d`)", "MAX(`d`)")
	}
	q := "SELECT " + strings.Join(fields, ", ") + " FROM `" + table + "`"
	values := make([]sql.NullString, len(fields))
	dest := make([]any, len(fields))
	for i := range values {
		dest[i] = &values[i]
	}
	if err := db.QueryRowContext(ctx, q).Scan(dest...); err != nil {
		return err
	}
	labels := []string{"transfer_min", "transfer_max", "transfer_avg"}
	if _, ok := cols["u"]; ok {
		labels = append(labels, "upload_min", "upload_max")
	}
	if _, ok := cols["d"]; ok {
		labels = append(labels, "download_min", "download_max")
	}
	fmt.Println("source_traffic_stats:")
	for i, label := range labels {
		if values[i].Valid {
			fmt.Printf("  %s: %s\n", label, values[i].String)
		} else {
			fmt.Printf("  %s: \n", label)
		}
	}
	return nil
}

func printSourcePlanExpiryStats(ctx context.Context, db *sql.DB, table string, cols map[string]columnMeta) error {
	if _, ok := cols["expired_at"]; !ok {
		return nil
	}
	q := "SELECT COALESCE(CAST(`plan_id` AS CHAR), ''), " +
		"SUM(CASE WHEN `expired_at` IS NULL OR `expired_at` = 0 THEN 1 ELSE 0 END), " +
		"SUM(CASE WHEN `expired_at` IS NOT NULL AND `expired_at` != 0 AND FROM_UNIXTIME(`expired_at`) <= UTC_TIMESTAMP() THEN 1 ELSE 0 END), " +
		"SUM(CASE WHEN `expired_at` IS NOT NULL AND `expired_at` != 0 AND FROM_UNIXTIME(`expired_at`) > UTC_TIMESTAMP() THEN 1 ELSE 0 END) " +
		"FROM `" + table + "` GROUP BY `plan_id` ORDER BY `plan_id` ASC"
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return err
	}
	defer rows.Close()
	fmt.Println("source_plan_expiry_distribution:")
	for rows.Next() {
		var planID string
		var noExpiry, expired, valid sql.NullInt64
		if err := rows.Scan(&planID, &noExpiry, &expired, &valid); err != nil {
			return err
		}
		if planID == "" {
			planID = "(null)"
		}
		fmt.Printf("  %s | no_expiry=%d | expired=%d | valid=%d\n", planID, nullInt(noExpiry), nullInt(expired), nullInt(valid))
	}
	return rows.Err()
}

func printSourcePasswordStats(ctx context.Context, db *sql.DB, table string, cols map[string]columnMeta) error {
	passwordCol, ok := pickColumn(cols, "password", "password_hash", "passwd")
	if !ok {
		return nil
	}
	q := "SELECT LEFT(`" + passwordCol + "`, 4), COUNT(*) FROM `" + table + "` GROUP BY LEFT(`" + passwordCol + "`, 4) ORDER BY COUNT(*) DESC"
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return err
	}
	defer rows.Close()
	fmt.Println("source_password_hash_prefixes:")
	for rows.Next() {
		var prefix string
		var count int64
		if err := rows.Scan(&prefix, &count); err != nil {
			return err
		}
		if prefix == "" {
			prefix = "(empty)"
		}
		fmt.Printf("  %s: %d\n", prefix, count)
	}
	return rows.Err()
}

func nullInt(v sql.NullInt64) int64 {
	if !v.Valid {
		return 0
	}
	return v.Int64
}

func printPlanRows(ctx context.Context, db *sql.DB, table, prefix string) error {
	cols, err := loadColumns(ctx, db, table)
	if err != nil {
		return err
	}
	if _, ok := cols["id"]; !ok {
		return nil
	}
	if _, ok := cols["name"]; !ok {
		return nil
	}
	fields := []string{"`id`", "`name`"}
	if _, ok := cols["transfer_enable"]; ok {
		fields = append(fields, "`transfer_enable`")
	} else if _, ok := cols["traffic_limit"]; ok {
		fields = append(fields, "`traffic_limit`")
	}
	if _, ok := cols["month_price"]; ok {
		fields = append(fields, "`month_price`")
	} else if _, ok := cols["price"]; ok {
		fields = append(fields, "`price`")
	}
	if _, ok := cols["device_limit"]; ok {
		fields = append(fields, "`device_limit`")
	}
	if _, ok := cols["status"]; ok {
		fields = append(fields, "`status`")
	}
	q := "SELECT " + strings.Join(fields, ", ") + " FROM `" + table + "` ORDER BY `id` ASC"
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		values := make([]sql.NullString, len(fields))
		dest := make([]any, len(fields))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return err
		}
		parts := make([]string, 0, len(values))
		for _, value := range values {
			if value.Valid {
				parts = append(parts, value.String)
			} else {
				parts = append(parts, "")
			}
		}
		fmt.Println(prefix + strings.Join(parts, " | "))
	}
	return rows.Err()
}

func parsePlanMap(raw string) (map[string]int64, error) {
	out := map[string]int64{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out, nil
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pair := strings.SplitN(part, ":", 2)
		if len(pair) != 2 {
			return nil, fmt.Errorf("invalid plan-map entry %q", part)
		}
		target, err := strconv.ParseInt(strings.TrimSpace(pair[1]), 10, 64)
		if err != nil || target <= 0 {
			return nil, fmt.Errorf("invalid target plan id in %q", part)
		}
		out[strings.TrimSpace(pair[0])] = target
	}
	return out, nil
}

func buildAutoPlanMap(ctx context.Context, source, target *sql.DB) (map[string]int64, error) {
	sourcePlanTable, err := detectTable(ctx, source, []string{"v2_plan", "plans", "plan"})
	if err != nil {
		return map[string]int64{}, nil
	}
	sourceCols, err := loadColumns(ctx, source, sourcePlanTable)
	if err != nil {
		return nil, err
	}
	if _, ok := sourceCols["id"]; !ok {
		return map[string]int64{}, nil
	}
	if _, ok := sourceCols["name"]; !ok {
		return map[string]int64{}, nil
	}

	targetPlans := map[string]int64{}
	rows, err := target.QueryContext(ctx, `SELECT id, name FROM plans`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		targetPlans[normalizePlanName(name)] = id
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := map[string]int64{}
	q := fmt.Sprintf("SELECT `id`, `name` FROM `%s`", sourcePlanTable)
	srcRows, err := source.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer srcRows.Close()
	for srcRows.Next() {
		var id, name string
		if err := srcRows.Scan(&id, &name); err != nil {
			return nil, err
		}
		if targetID, ok := targetPlans[normalizePlanName(name)]; ok {
			out[strings.TrimSpace(id)] = targetID
		}
	}
	return out, srcRows.Err()
}

func normalizePlanName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func ensureTargetPlanExists(ctx context.Context, db *sql.DB, id int64) error {
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM plans WHERE id = ?`, id).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("target plan id %d does not exist", id)
	}
	return nil
}

func loadExistingEmails(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT email FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		out[strings.ToLower(strings.TrimSpace(email))] = true
	}
	return out, rows.Err()
}

func loadTargetPlanDeviceLimits(ctx context.Context, db *sql.DB) (map[int64]int, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, device_limit FROM plans`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int{}
	for rows.Next() {
		var id int64
		var limit int
		if err := rows.Scan(&id, &limit); err != nil {
			return nil, err
		}
		out[id] = limit
	}
	return out, rows.Err()
}

func loadActiveNodes(ctx context.Context, db *sql.DB) ([]nodeRow, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, protocol FROM nodes WHERE status = 'active' ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []nodeRow
	for rows.Next() {
		var n nodeRow
		if err := rows.Scan(&n.ID, &n.Protocol); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func loadSourceUsers(ctx context.Context, db *sql.DB, table string, cols map[string]columnMeta, limit int, copyBalance, balanceCents bool) ([]sourceUser, error) {
	emailCol, ok := pickColumn(cols, "email")
	if !ok {
		return nil, errors.New("source user table missing email column")
	}
	passwordCol, ok := pickColumn(cols, "password", "password_hash", "passwd")
	if !ok {
		return nil, errors.New("source user table missing password column")
	}

	wanted := []string{
		"id", emailCol, passwordCol, "token", "uuid", "plan_id", "transfer_enable", "u", "d",
		"expired_at", "created_at", "updated_at", "banned", "status", "enable", "is_active",
		"balance", "device_limit",
	}
	selectCols := existingSelectColumns(cols, wanted)
	if len(selectCols) == 0 {
		return nil, errors.New("source user select columns empty")
	}
	q := "SELECT " + quoteColumns(selectCols) + " FROM `" + table + "`"
	if _, ok := cols["id"]; ok {
		q += " ORDER BY `id` ASC"
	}
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []sourceUser
	for rows.Next() {
		values := make([]sql.NullString, len(selectCols))
		dest := make([]any, len(selectCols))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		row := map[string]string{}
		for i, col := range selectCols {
			if values[i].Valid {
				row[strings.ToLower(col)] = values[i].String
			}
		}
		out = append(out, sourceUserFromRow(row, copyBalance, balanceCents))
	}
	return out, rows.Err()
}

func pickColumn(cols map[string]columnMeta, names ...string) (string, bool) {
	for _, name := range names {
		if _, ok := cols[strings.ToLower(name)]; ok {
			return strings.ToLower(name), true
		}
	}
	return "", false
}

func existingSelectColumns(cols map[string]columnMeta, wanted []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, name := range wanted {
		name = strings.ToLower(name)
		if seen[name] {
			continue
		}
		if _, ok := cols[name]; ok {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func quoteColumns(cols []string) string {
	parts := make([]string, 0, len(cols))
	for _, c := range cols {
		parts = append(parts, "`"+strings.ReplaceAll(c, "`", "``")+"`")
	}
	return strings.Join(parts, ", ")
}

func sourceUserFromRow(row map[string]string, copyBalance, balanceCents bool) sourceUser {
	now := time.Now().UTC()
	expiredAt := parseOptionalTime(first(row, "expired_at"))
	createdAt := parseTimeOrDefault(first(row, "created_at"), now)
	updatedAt := parseTimeOrDefault(first(row, "updated_at"), createdAt)
	upload := parseInt64(first(row, "u"))
	download := parseInt64(first(row, "d"))
	limit := parseInt64(first(row, "transfer_enable"))
	used := upload + download

	balance := "0.00"
	if copyBalance {
		balance = normalizeBalance(first(row, "balance"), balanceCents)
	}

	return sourceUser{
		SourceID:     first(row, "id"),
		Email:        strings.ToLower(strings.TrimSpace(first(row, "email"))),
		PasswordHash: strings.TrimSpace(first(row, "password", "password_hash", "passwd")),
		Balance:      balance,
		PlanID:       strings.TrimSpace(first(row, "plan_id")),
		DeviceLimit:  int(parseInt64(first(row, "device_limit"))),
		ExpiredAt:    expiredAt,
		TrafficLimit: limit,
		TrafficUsed:  used,
		Upload:       upload,
		Download:     download,
		Status:       sourceStatus(row),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

func first(row map[string]string, names ...string) string {
	for _, name := range names {
		if v, ok := row[strings.ToLower(name)]; ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseOptionalTime(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" || strings.EqualFold(raw, "null") {
		return nil
	}
	if n, ok := parseNumber(raw); ok {
		if n <= 0 {
			return nil
		}
		if n > 1000000000000 {
			n = n / 1000
		}
		t := time.Unix(n, 0).UTC()
		return &t
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			u := t.UTC()
			return &u
		}
	}
	return nil
}

func parseTimeOrDefault(raw string, fallback time.Time) time.Time {
	if t := parseOptionalTime(raw); t != nil {
		return *t
	}
	return fallback
}

func parseNumber(raw string) (int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	if strings.Contains(raw, ".") {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return 0, false
		}
		return int64(math.Round(f)), true
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	return n, err == nil
}

func parseInt64(raw string) int64 {
	n, _ := parseNumber(raw)
	if n < 0 {
		return 0
	}
	return n
}

func normalizeBalance(raw string, cents bool) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "0.00"
	}
	if strings.Contains(raw, ".") {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return "0.00"
		}
		return fmt.Sprintf("%.2f", f)
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return "0.00"
	}
	if cents {
		return fmt.Sprintf("%d.%02d", n/100, abs64(n%100))
	}
	return fmt.Sprintf("%d.00", n)
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func sourceStatus(row map[string]string) string {
	if isTruthy(first(row, "banned")) {
		return "disabled"
	}
	status := strings.ToLower(first(row, "status"))
	switch status {
	case "disabled", "disable", "inactive", "banned", "ban", "blocked", "0":
		return "disabled"
	}
	enable := strings.TrimSpace(first(row, "enable", "is_active"))
	if enable != "" && !isTruthy(enable) {
		return "disabled"
	}
	return "active"
}

func isTruthy(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "active", "enabled", "enable":
		return true
	default:
		return false
	}
}

func resolveTargetPlanID(sourcePlanID string, explicit, auto map[string]int64, defaultPlanID int64) *int64 {
	sourcePlanID = strings.TrimSpace(sourcePlanID)
	if sourcePlanID != "" {
		if v, ok := explicit[sourcePlanID]; ok {
			return &v
		}
		if v, ok := auto[sourcePlanID]; ok {
			return &v
		}
	}
	if defaultPlanID > 0 {
		return &defaultPlanID
	}
	return nil
}

func provisionableSourceUser(u sourceUser, now time.Time) bool {
	if u.Status != "active" {
		return false
	}
	if u.ExpiredAt != nil && !u.ExpiredAt.After(now) {
		return false
	}
	if u.TrafficLimit > 0 && u.TrafficUsed >= u.TrafficLimit {
		return false
	}
	return true
}

func insertTargetUser(ctx context.Context, tx *sql.Tx, u sourceUser, planID *int64) (int64, error) {
	var planArg any
	if planID != nil {
		planArg = *planID
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO users(email, password_hash, balance, plan_id, plan_period, expired_at,
		traffic_limit, traffic_used, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'monthly', ?, ?, ?, ?, ?, ?)`,
		u.Email, u.PasswordHash, u.Balance, planArg, u.ExpiredAt,
		u.TrafficLimit, u.TrafficUsed, u.Status, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func insertSubToken(ctx context.Context, tx *sql.Tx, userID int64, token string, createdAt time.Time) error {
	hash := sha256.Sum256([]byte(token))
	_, err := tx.ExecContext(ctx, `INSERT INTO subscription_tokens(user_id, token, token_hash, status, created_at, updated_at)
		VALUES (?, ?, ?, 'active', ?, ?)`,
		userID, token, hex.EncodeToString(hash[:]), createdAt, createdAt)
	return err
}

func upsertTrafficSnapshot(ctx context.Context, tx *sql.Tx, userID int64, u sourceUser) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO user_traffic_snapshots(user_id, upload_total, download_total, total_used, traffic_limit, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE upload_total = VALUES(upload_total), download_total = VALUES(download_total),
			total_used = VALUES(total_used), traffic_limit = VALUES(traffic_limit), updated_at = VALUES(updated_at)`,
		userID, u.Upload, u.Download, u.TrafficUsed, u.TrafficLimit, u.UpdatedAt)
	return err
}

func insertNodeUser(ctx context.Context, tx *sql.Tx, userID int64, n nodeRow, clientID string, deviceLimit int, createdAt time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO node_users(user_id, node_id, client_id, protocol, enabled, upload, download,
		speed_limit, device_limit, created_at, updated_at)
		VALUES (?, ?, ?, ?, 1, 0, 0, 0, ?, ?, ?)`,
		userID, n.ID, clientID, n.Protocol, deviceLimit, createdAt, createdAt)
	return err
}

func randomHex(bytesN int) (string, error) {
	buf := make([]byte, bytesN)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func newClientID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func redactEmail(email string) string {
	email = strings.TrimSpace(email)
	at := strings.IndexByte(email, '@')
	if at <= 1 {
		return "***"
	}
	return email[:1] + "***" + email[at:]
}

func printReport(r report, executed bool) {
	mode := "DRY-RUN"
	if executed {
		mode = "EXECUTE"
	}
	fmt.Println("mode:", mode)
	fmt.Println("source_table:", r.SourceTable)
	fmt.Println("source_total:", r.SourceTotal)
	fmt.Println("auto_plan_matches:", r.AutoPlanMatches)
	fmt.Println("active_target_nodes:", r.ActiveNodes)
	fmt.Println("would_insert:", r.WouldInsert)
	fmt.Println("inserted:", r.Inserted)
	fmt.Println("with_target_plan:", r.WithTargetPlan)
	fmt.Println("without_target_plan:", r.WithoutTargetPlan)
	fmt.Println("active_users:", r.ActiveUsers)
	fmt.Println("disabled_users:", r.DisabledUsers)
	fmt.Println("provisionable_users:", r.Provisionable)
	fmt.Println("expired_users:", r.ExpiredUsers)
	fmt.Println("no_expiry_users:", r.NoExpiryUsers)
	fmt.Println("skipped_existing_email:", r.SkippedExisting)
	fmt.Println("skipped_bad_email:", r.SkippedBadEmail)
	fmt.Println("skipped_no_password:", r.SkippedNoPassword)
	fmt.Println("skipped_errors:", r.SkippedErrors)
	if !executed {
		fmt.Println("note: dry-run only; add -execute after backup and confirmation")
	}
}
