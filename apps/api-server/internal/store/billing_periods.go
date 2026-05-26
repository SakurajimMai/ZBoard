package store

import "strings"

func NormalizeBillingPeriod(period string) string {
	switch strings.ToLower(strings.TrimSpace(period)) {
	case BillingPeriodQuarterly:
		return BillingPeriodQuarterly
	case BillingPeriodYearly:
		return BillingPeriodYearly
	default:
		return BillingPeriodMonthly
	}
}

func BillingPeriodMultiplier(period string) int64 {
	switch NormalizeBillingPeriod(period) {
	case BillingPeriodQuarterly:
		return 3
	case BillingPeriodYearly:
		return 12
	default:
		return 1
	}
}

func PlanDurationDays(plan *Plan, period string) int {
	days := plan.DurationDays
	if days <= 0 {
		days = 30
	}
	return days * int(BillingPeriodMultiplier(period))
}

func PlanTrafficLimit(plan *Plan, period string) int64 {
	if plan.TrafficLimit <= 0 {
		return 0
	}
	return plan.TrafficLimit * BillingPeriodMultiplier(period)
}
