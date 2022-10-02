package setlogs

import (
	"encoding/json"
	"io"
)

func ReadJSON(r io.Reader) SetLog {
	var setlog SetLog
	json.NewDecoder(r).Decode(&setlog)
	return setlog
}

func WriteJSON(w io.Writer, setlog SetLog) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(setlog)
}
