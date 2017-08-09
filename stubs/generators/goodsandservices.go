package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
)

type codeList struct {
	ID    string               `json:"id"`
	Name  string               `json:"name"`
	Codes map[string]codeEntry `json:"codes"`
}

type codeEntry struct {
	ID    string `json:"id"`
	Code  string `json:"code"`
	Label string `json:"label"`
}

func main() {
	f, _ := os.Open("goodsandservices.csv")
	csvr := csv.NewReader(f)
	defer f.Close()

	recs, _ := csvr.ReadAll()

	o := make(map[string]codeEntry)
	for _, v := range recs {
		id := uuid.New().String()
		o[id] = codeEntry{
			ID:    id,
			Code:  v[0],
			Label: v[1],
		}
	}

	cl := codeList{
		ID:    uuid.New().String(),
		Name:  "Goods and Services Indices",
		Codes: o,
	}

	b, _ := json.MarshalIndent(&cl, "", " ")
	fmt.Println(string(b))
}
