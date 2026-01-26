package orm

// Pageable represents pagination parameters
type Pageable struct {
	Page      int    // Zero-based page number
	Size      int    // Number of items per page
	SortBy    string // Property to sort by
	Direction string // ASC or DESC
}

// NewPageable creates a new Pageable with default values
func NewPageable(page, size int) *Pageable {
	return &Pageable{
		Page:      page,
		Size:      size,
		Direction: "ASC",
	}
}

// WithSort sets the sort parameters
func (p *Pageable) WithSort(sortBy, direction string) *Pageable {
	p.SortBy = sortBy
	p.Direction = direction
	return p
}

// Next returns a Pageable for the next page
func (p *Pageable) Next() *Pageable {
	return &Pageable{
		Page:      p.Page + 1,
		Size:      p.Size,
		SortBy:    p.SortBy,
		Direction: p.Direction,
	}
}

// Previous returns a Pageable for the previous page
func (p *Pageable) Previous() *Pageable {
	page := p.Page - 1
	if page < 0 {
		page = 0
	}
	return &Pageable{
		Page:      page,
		Size:      p.Size,
		SortBy:    p.SortBy,
		Direction: p.Direction,
	}
}

// Offset returns the number of items to skip
func (p *Pageable) Offset() int {
	return p.Page * p.Size
}

// Page represents a page of results
type Page[T any] struct {
	Content       []*T  // The page content
	PageNumber    int   // Current page number (zero-based)
	PageSize      int   // Number of items per page
	TotalElements int64 // Total number of elements
	TotalPages    int   // Total number of pages
}

// NewPage creates a new Page instance
func NewPage[T any](content []*T, pageable *Pageable, totalElements int64) *Page[T] {
	totalPages := int(totalElements) / pageable.Size
	if int(totalElements)%pageable.Size > 0 {
		totalPages++
	}

	return &Page[T]{
		Content:       content,
		PageNumber:    pageable.Page,
		PageSize:      pageable.Size,
		TotalElements: totalElements,
		TotalPages:    totalPages,
	}
}

// HasNext returns true if there is a next page
func (p *Page[T]) HasNext() bool {
	return p.PageNumber+1 < p.TotalPages
}

// HasPrevious returns true if there is a previous page
func (p *Page[T]) HasPrevious() bool {
	return p.PageNumber > 0
}

// IsFirst returns true if this is the first page
func (p *Page[T]) IsFirst() bool {
	return p.PageNumber == 0
}

// IsLast returns true if this is the last page
func (p *Page[T]) IsLast() bool {
	return p.PageNumber+1 >= p.TotalPages
}

// NumberOfElements returns the number of elements in this page
func (p *Page[T]) NumberOfElements() int {
	return len(p.Content)
}
