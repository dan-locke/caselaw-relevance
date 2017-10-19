package main

import (
	"errors"
	"fmt"

	elastic "elastic-go"
)

type Decision struct {

	Id string `json:"id"`

	CaseName string `json:"case_name"`

	DateFiled string `json:"date_filed"`

	Html string `json:"html"`

	// PlainText string `json:"plain_text"`
}

// type ApiQueryResponse struct {
//
// 	Query string `json:"query"`
//
// 	TotalHits int64 `json:"total"`
//
// 	Hits []ApiCaseResponse `json:"hits"`
//
// }

type ApiCaseResponse struct {

	Score float64 `json:"score"`

	Id string `json:"id"`

	CaseName string `json:"case_name"`

	DateFiled string `json:"date_filed"`

	Html string `json:"html"`

	Relevance string `json:"relevance"`

	Stored bool `json:"stored"`

}

type ApiGetResponse Decision

func parseDecisionFromMap(m map[string]interface{}) (Decision, error) {
	dec := Decision{}
	var ok bool
	for k, v := range m {
		switch k {
			case "id":
				f, ok := v.(float64)
				dec.Id = fmt.Sprintf("%f", f)
				if !ok {
					return Decision{}, errors.New("Could not pass Id as int64")
				}
			case "case_name":
				dec.CaseName, ok = v.(string)
				if !ok {
					return Decision{}, errors.New("Could not pass Case name as string")
				}
			case "date_filed":
				dec.DateFiled, ok = v.(string)
				if !ok {
					return Decision{}, errors.New("Could not pass Date filed as string")
				}
			case "html":
				dec.Html, ok = v.(string)
				if !ok {
					return Decision{}, errors.New("Could not pass html as string")
				}
		}
	}
	return dec, nil
}

func elasticGetToApiResponse(s *elastic.GetResponse) (*ApiGetResponse, error) {
	res, err := parseDecisionFromMap(s.Source.(map[string]interface{}))
	if err != nil {
		return nil, err
	}
	r := ApiGetResponse(res)

	return &r, nil
}

func elasticSearchToApiCaseResponse(s *elastic.SearchResponse) ([]ApiCaseResponse, error) {
	res := make([]ApiCaseResponse, len(s.Hits.Hits))
	for i := range s.Hits.Hits {
		hit, err := parseDecisionFromMap(s.Hits.Hits[i].Source.(map[string]interface{}))
		if err != nil {
			return nil, err
		}
		res[i] = ApiCaseResponse {
			Score : s.Hits.Hits[i].Score,
			Id : s.Hits.Hits[i].Id,
			CaseName : hit.CaseName,
			DateFiled : hit.DateFiled,
			Html : hit.Html,
			// Excerpt : s.Hits.Hits[i].Highlights.Highlight,
		}
	}
	return res, nil
}

// func elasticSearchToApiQueryResponse(query []byte, s *elastic.SearchResponse) (*ApiQueryResponse, error) {
// 	res := ApiQueryResponse{
// 		Query : string(query),
// 		TotalHits: s.Hits.Total,
// 		Hits : make([]ApiCaseResponse, len(s.Hits.Hits)),
// 	}
// 	for i := range s.Hits.Hits {
// 		hit, err := parseDecisionFromMap(s.Hits.Hits[i].Source.(map[string]interface{}))
// 		if err != nil {
// 			return nil, err
// 		}
//
// 		res.Hits[i] = ApiCaseResponse {
// 			Score : s.Hits.Hits[i].Score,
// 			Id : s.Hits.Hits[i].Id,
// 			CaseName : hit.CaseName,
// 			DateFiled : hit.DateFiled,
// 			Html : hit.Html,
// 			Excerpt : s.Hits.Hits[i].Highlights.Highlight,
// 		}
//
// 	}
// 	return &res, nil
// }

// TODO - parsing structs from generic map[string]interefaces
// const trpTag string = "trp"

// type structFieldMap map[string]struct{
// 	Index int
// 	Type reflect.Type
// }
//
// // Adapted from https://stackoverflow.com/questions/26744873/converting-map-to-struct
// // changing map[string]interface{} to struct ... and encoding/json marshal
// func generateStructTagFieldMap(obj interface{}) (structFieldMap, error) {
// 	tagFieldIndex := map[string]struct{
// 		Index int
// 		Type reflect.Type
// 	}{}
// 	structVal := reflect.ValueOf(obj).Elem()
// 	// fmt.Println(structVal)
// 	for i := 0; i < structVal.NumField(); i++ {
// 		tagFieldIndex[structVal.Type().Field(i).Tag.Get("json")] = struct{Index int; Type reflect.Type}{
// 			Index: i,
// 			Type: structVal.Field(i).Type(),
// 		}
// 	}
// 	return tagFieldIndex, nil
// }
//
// func (s structFieldMap)structFromMap(obj map[string]interface{}, res interface{}) error {
// 	resStruct := reflect.ValueOf(res).Elem()
// 	for k, v := range obj {
// 		fmt.Println("Here -", k, v)
// 		if i, ok := s[k]; !ok {
// 			return fmt.Errorf("Error parsing to struct. No struct attribute %s", k)
// 		} else {
// 			resStructVal := resStruct.Field(i.Index)
// 			fmt.Println("Here", resStructVal.Type())
// 			if !resStructVal.CanSet() {
// 				return fmt.Errorf("Cannot set %s field value", k)
// 			}
// 			structFieldType := resStructVal.Type()
// 			val := reflect.ValueOf(v)
// 			if structFieldType != val.Type() {
// 				fmt.Println(val.Type())
// 				return fmt.Errorf("Provided value type didn't match obj field type")
// 			}
// 			resStructVal.Set(val)
// 		}
// 	}
// 	return nil
// }
