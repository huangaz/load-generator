package gen

import "errors"

type GoTickets interface {
	// get one ticket
	Take()

	// return one ticket
	Return()

	// is active
	Active() bool

	// total number of tickets
	Total() uint32

	// remain number of tickets
	Remainder() uint32
}

type myGoTickets struct {
	// total number of tickets
	total uint32

	// container of tickets
	ticketCh chan struct{}

	// is active
	active bool
}

func NewGoTickets(total uint32) (GoTickets, error) {
	if total == 0 {
		return nil, errors.New("invalid concurrency")
	}

	res := &myGoTickets{
		total:    total,
		ticketCh: make(chan struct{}, total),
		active:   true,
	}
	return res, nil
}

func (m *myGoTickets) Take() {
	m.ticketCh <- struct{}{}
}

func (m *myGoTickets) Return() {
	<-m.ticketCh
}

func (m *myGoTickets) Active() bool {
	return m.active
}

func (m *myGoTickets) Total() uint32 {
	return m.total
}

func (m *myGoTickets) Remainder() uint32 {
	return m.total - uint32(len(m.ticketCh))
}
