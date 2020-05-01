/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package utils

import (
	"net"
	"strings"
)

// NewOrderedNavigableMap initializates a structure of OrderedNavigableMap with a NavigableMap2
func NewOrderedNavigableMap() *OrderedNavigableMap {
	return &OrderedNavigableMap{
		nm:       NavigableMap2{},
		orderIdx: NewPathItemList(),
		orderRef: make(map[string][]*PathItemElement),
	}
}

// OrderedNavigableMap is the same as NavigableMap2 but keeps the order of fields
type OrderedNavigableMap struct {
	nm       NMInterface
	orderIdx *PathItemList
	orderRef map[string][]*PathItemElement
}

// String returns the map as json string
func (onm *OrderedNavigableMap) String() string {
	return onm.nm.String()
}

// GetFirstElement returns the first element from the order
func (onm *OrderedNavigableMap) GetFirstElement() *PathItemElement {
	return onm.orderIdx.Front()
}

// Interface returns navigble map that's inside
func (onm *OrderedNavigableMap) Interface() interface{} {
	return onm.nm
}

// Field returns the item on the given path
func (onm *OrderedNavigableMap) Field(fldPath PathItems) (val NMInterface, err error) {
	return onm.nm.Field(fldPath)
}

// Type returns the type of the NM map
func (onm *OrderedNavigableMap) Type() NMType {
	return onm.nm.Type()
}

// Empty returns true if the NM is empty(no data)
func (onm *OrderedNavigableMap) Empty() bool {
	return onm.nm.Empty()
}

// Remove removes the item for the given path and updates the order
func (onm *OrderedNavigableMap) Remove(path FullPath) (err error) {
	if err = onm.nm.Remove(path.PathItems); err != nil {
		return
	}
	onm.removePath(path.Path, path.PathItems[len(path.PathItems)-1].Index)
	if path.PathItems[len(path.PathItems)-1].Index != nil {
		return ErrNotImplemented // for the momment we can't remove only a specific element
	}
	return
}

// Set sets the value at the given path
// this is the old to be capable of  building the code without updating all the code
// will be replaced with Set2 after we decide that is the optimal solution
func (onm *OrderedNavigableMap) Set(fldPath PathItems, val NMInterface) (addedNew bool, err error) {
	return onm.Set2(&FullPath{PathItems: fldPath, Path: fldPath.String()}, val)
}

// Set2 sets the value at the given path
// this used with full path and the processed path to not calculate them for every set
func (onm *OrderedNavigableMap) Set2(fullPath *FullPath, val NMInterface) (addedNew bool, err error) {
	if len(fullPath.PathItems) == 0 {
		return false, ErrWrongPath
	}
	if addedNew, err = onm.nm.Set(fullPath.PathItems, val); err != nil {
		return
	}

	var pathItmsSet []PathItems // can be multiples if we need to inflate due to missing Index in slice set
	var nonIndexedSlcPath bool
	if val.Type() == NMSliceType && fullPath.PathItems[len(fullPath.PathItems)-1].Index == nil { // special case when we overwrite with a slice without specifying indexes
		nonIndexedSlcPath = true
		pathItmsSet = make([]PathItems, len(*val.(*NMSlice)))
		for i := range *val.(*NMSlice) {
			pathItms := fullPath.PathItems.Clone()
			pathItms[len(pathItms)-1].Index = IntPointer(i)
			pathItmsSet[i] = pathItms
		}
	} else {
		pathItmsSet = []PathItems{fullPath.PathItems.Clone()}
	}
	path := stripIdxFromLastPathElm(fullPath.Path)
	if !addedNew && nonIndexedSlcPath { // cleanup old references since the value is being overwritten
		for idxPath, slcIdx := range onm.orderRef {
			if !strings.HasPrefix(idxPath, path) {
				continue
			}
			for _, el := range slcIdx {
				onm.orderIdx.Remove(el)
			}
			delete(onm.orderRef, path)
		}
	}
	_, hasRef := onm.orderRef[path]
	for _, pathItms := range pathItmsSet {
		if addedNew || !hasRef {
			onm.orderRef[path] = append(onm.orderRef[path], onm.orderIdx.PushBack(pathItms))
		} else {
			onm.orderIdx.MoveToBack(onm.orderRef[path][len(onm.orderRef[path])-1])
		}
	}
	return true, nil
}

// removePath removes any reference to the given path from order
func (onm *OrderedNavigableMap) removePath(path string, indx *int) {
	if indx == nil {
		for _, el := range onm.orderRef[path] {
			onm.orderIdx.Remove(el)
		}
		delete(onm.orderRef, path)
		return
	}
	i := 0
	for ; i < len(onm.orderRef[path]); i++ {
		path := onm.orderRef[path][i].Value
		if *path[len(path)-1].Index == *indx {
			break
		}
	}
	if i < len(onm.orderRef[path]) {
		onm.orderIdx.Remove(onm.orderRef[path][i])
		onm.orderRef[path][i] = nil
		onm.orderRef[path] = onm.orderRef[path][:i+copy(onm.orderRef[path][:i], onm.orderRef[path][i+1:])]
	}
}

// GetField the same as Field but for one level deep
// used to implement NM interface
func (onm *OrderedNavigableMap) GetField(path PathItem) (val NMInterface, err error) {
	return onm.nm.GetField(path)
}

// Len returns the lenght of the map
func (onm OrderedNavigableMap) Len() int {
	return onm.nm.Len()
}

// FieldAsString returns thevalue from path as string
func (onm *OrderedNavigableMap) FieldAsString(fldPath []string) (str string, err error) {
	var val NMInterface
	val, err = onm.nm.Field(NewPathToItem(fldPath))
	if err != nil {
		return
	}
	return IfaceAsString(val.Interface()), nil
}

// FieldAsInterface returns the interface at the path
func (onm *OrderedNavigableMap) FieldAsInterface(fldPath []string) (str interface{}, err error) {
	var val NMInterface
	val, err = onm.nm.Field(NewPathToItem(fldPath))
	if err != nil {
		return
	}
	return val.Interface(), nil
}

// RemoteHost is part of dataStorage interface
func (OrderedNavigableMap) RemoteHost() net.Addr {
	return LocalAddr()
}

// GetOrder returns the elements order as a slice
// use this only for testing
func (onm *OrderedNavigableMap) GetOrder() (order []PathItems) {
	for el := onm.GetFirstElement(); el != nil; el = el.Next() {
		order = append(order, el.Value)
	}
	return
}
