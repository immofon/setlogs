package setlogs

const ID string = "@id"
const TypeBase string = "base"
const TypeMutate string = "mutate"
const TypeSet string = "set"

type Record map[string]string
type Set map[string]bool

func NewRecord() Record {
	return make(Record)
}

func NewSet() Set {
	return make(Set)
}

type SetLog struct {
	Type    string
	Records []Record
	Comment string
}

func New(log_type string) SetLog {
	if log_type != TypeBase && log_type != TypeMutate && log_type != TypeSet {
		panic("the log_type MUST be base or mutate or set!")
	}
	return SetLog{
		Type:    log_type,
		Records: make([]Record, 0),
		Comment: "",
	}
}

func (l SetLog) Check() bool {
	// the Type of SetLog MUST be Base or Mutate or Set.
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
	for i, record := range l.Records {
		l.Records[i] = update(record)
	}
}

// Merge log typed mutate to l typed base.
func (l *SetLog) Merge(log SetLog) {
	if l.Type != TypeBase {
		panic("You can only Merge SetLog to a SetLog [[typed base]]!")
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
