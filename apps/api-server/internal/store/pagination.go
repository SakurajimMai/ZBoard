package store

type PageParams struct {
	Page     int
	PageSize int
}

func NormalizePage(p PageParams) PageParams {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return p
}

func (p PageParams) Offset() int {
	p = NormalizePage(p)
	return (p.Page - 1) * p.PageSize
}
