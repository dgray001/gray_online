package goldset

import (
	"encoding/json"
	"net/http"
)

func (s *Store) labelHandler(w http.ResponseWriter, r *http.Request) {
	var label Label
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&label); err != nil {
		fail(w, 400, err)
		return
	}
	if err := s.SaveLabel(r.PathValue("id"), label); err != nil {
		fail(w, 400, err)
		return
	}
	item, _ := s.Get(r.PathValue("id"))
	respond(w, 200, item)
}

func (s *Store) generateHandler(w http.ResponseWriter, r *http.Request) {
	var request GenerateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&request); err != nil {
		fail(w, 400, err)
		return
	}
	items, err := Generate(request)
	if err != nil {
		fail(w, 400, err)
		return
	}
	added, err := s.Add(items)
	if err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]int{"added": added})
}
