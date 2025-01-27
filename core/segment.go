/*******************************************************************************
 * Audio Morse Decoder 'CW4ISR' (forked from Project ZyLO since 2023 July 15th)
 * Released under the MIT License (or GPL v3 until 2021 Oct 28th) (see LICENSE)
 * Univ. Tokyo Amateur Radio Club Development Task Force (https://nextzlog.dev)
*******************************************************************************/

package core

type Segment struct {
	Class bool
	Since int
	Until int
	Width float64
	Level float64
}

func (m *Classes) Segments(first int) (result []Segment) {
	since := 0
	for until, value := range m.X {
		if class := m.Class(value); first != class {
			result = append(result, Segment{
				Class: first == 1,
				Since: since,
				Until: until,
				Width: float64(until - since),
				Level: med64(m.X[since:until]),
			})
			since = until
			first = class
		}

		if until == len(m.X) -1 && since != until {
			result = append(result, Segment{
				Class: first == 1,
				Since: since,
				Until: until,
				Width: float64(until - since),
				Level: med64(m.X[since:until]),
			})
		}
	}

	if len(result) > 2 {
		return result[1:]
	} else {
		return nil
	}
}

func (m *Classes) Code(segments []Segment) (code string) {
	for _, s := range segments {
		if s.Class {
			switch m.Class(s.Width) {
			case 0:
				code += "."
			case 1:
				code += "_"
			}
		} else {
			switch m.Extra(s.Width) {
			case 0:
				code += ""
			case 1:
				code += " "
			default:
				code += " ; "
			}
		}
	}
	return
}
