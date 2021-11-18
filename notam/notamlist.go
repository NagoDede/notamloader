package notam

import (

)

type NotamList struct {
	Data map[string]*Notam
}

func NewNotamList() *NotamList {
	list := &(NotamList{})
	list.Data = make(map[string]*Notam) 
	return list
}

