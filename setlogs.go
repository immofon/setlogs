package setlogs

import (
	"io"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
)

const ID string = "@id"
const DELETE string = "@delete"

type Type string

const TypeEmpty Type = ""
const TypeBase Type = "base"
const TypeMutate Type = "mutate"
const TypeSet Type = "set"

type Record map[string]string
type Set map[string]bool

func NewRecord() Record {
	return make(Record)
}

func (r Record) SetDelete(del bool) {
	if del {
		r[DELETE] = "true"
	} else {
		delete(r, DELETE)
	}
}

func NewSet() Set {
	return make(Set)
}

type SetLog struct {
	Type    Type
	Records []Record
	Comment string
}

func New(log_type Type) SetLog {
	if log_type != TypeBase && log_type != TypeMutate && log_type != TypeSet && log_type != TypeEmpty {
		panic("the log_type MUST be base or mutate or set or empty!")
	}
	return SetLog{
		Type:    log_type,
		Records: make([]Record, 0),
		Comment: "",
	}
}

func (l SetLog) Check() bool {
	if l.Type == TypeEmpty {
		return true
	}

	// the Type of SetLog MUST be Base or Mutate or Set or Empty.
	if l.Type != TypeBase && l.Type != TypeMutate && l.Type != TypeSet {
		return false
	}

	// each Recard MUST have record key valued ID.
	for _, record := range l.Records {
		if record == nil {
			return false
		}
		if record[ID] == "" {
			return false
		}
	}

	// each ID CAN occur almost once in SetType typed base or set
	if l.Type == TypeBase || l.Type == TypeSet {
		id_occured := NewSet()
		for _, record := range l.Records {
			if id_occured[record[ID]] {
				return false
			}
			id_occured[record[ID]] = true
		}
	}

	return true
}

func (l SetLog) Set(key string) Set {
	set := NewSet()
	for _, record := range l.Records {
		v := record[key]
		if v != "" {
			set[v] = true
		}
	}

	return set
}

func Filter(records []Record, exist func(r Record) bool) []Record {
	var rs []Record
	for _, r := range records {
		if exist(r) {
			rs = append(rs, r)
		}
	}
	return rs
}
func FilterByID(records []Record, id_set Set) []Record {
	return Filter(records, func(r Record) bool {
		return id_set[r[ID]]
	})
}

func (l SetLog) Filter(exist func(r Record) bool) []Record {
	return Filter(l.Records, exist)
}

func (l SetLog) FilterByID(id_set Set) []Record {
	return FilterByID(l.Records, id_set)
}

func (l *SetLog) Update(update func(r Record) Record) {
	new_records := make([]Record, 0, len(l.Records))
	for _, record := range l.Records {
		r := update(record)
		if r[DELETE] == "" {
			new_records = append(new_records, r)
		}
	}
	l.Records = new_records
}

// Merge log typed mutate to l typed base.
func (l *SetLog) Merge(log SetLog) {
	if l.Type == TypeEmpty {
		if log.Type == TypeBase {
			l.Type = log.Type
			l.Records = log.Records
			l.Comment = log.Comment
			return
		} else {
			panic("You cannot Merge non-Base SetLog to Empty SetLog")
		}
	}
	if l.Type != TypeBase {
		panic("You can only Merge SetLog to a SetLog [[typed base]]!")
	}

	if log.Type == TypeEmpty || log.Type == TypeSet {
		return
	}

	if log.Type != TypeMutate {
		panic("You can only Merge SetLog [[typed mutate]] to a SetLog typed base!")
	}

	id_set_of_l := l.Set(ID)
	for _, new_record := range log.Records {
		if id_set_of_l[new_record[ID]] {
			l.Update(func(r Record) Record {
				if r[ID] != new_record[ID] {
					return r
				}
				for k, v := range new_record {
					if v != "" {
						r[k] = v
					} else {
						delete(r, k)
					}
				}
				return r
			})
		} else { // ID of new_record is not occur in l.Set(ID).
			l.Records = append(l.Records, new_record)
		}
	}
}

func (l *SetLog) AppendRecords(records ...Record) {
	for _, r := range records {
		l.Records = append(l.Records, r)
	}
}

func (l SetLog) Keys() []string {
	keys_map := make(map[string]bool)
	for _, r := range l.Records {
		for k, _ := range r {
			keys_map[k] = true
		}
	}
	keys := make([]string, 0, len(keys_map))
	for k, _ := range keys_map {
		keys = append(keys, k)
	}

	sort.Sort(sort.StringSlice(keys))
	return keys
}

func (l SetLog) TableFprintln(w io.Writer) {
	header := l.Keys()

	table := tablewriter.NewWriter(w)
	table.SetHeader(header)
	table.SetAutoFormatHeaders(false)

	header_colors := make([]tablewriter.Colors, len(header))
	for i, key := range header {
		if strings.HasPrefix(key, "@") {
			header_colors[i] = tablewriter.Colors{tablewriter.FgHiYellowColor}
		}
	}
	table.SetHeaderColor(header_colors...)

	for _, r := range l.Records {
		row := make([]string, len(header))
		for j, k := range header {
			row[j] = r[k]
		}
		table.Append(row)
	}

	column_colors := make([]tablewriter.Colors, len(header))
	for i, key := range header {
		if key == "@id" {
			column_colors[i] = tablewriter.Colors{tablewriter.Bold}
		}
	}
	table.SetColumnColor(column_colors...)

	table.Render()
}
