// SPDX-License-Identifier: Apache-2.0
//
// Copyright (C) 2021 Renesas Electronics Corporation.
// Copyright (C) 2021 EPAM Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dataadaptertest

import (
	"reflect"
	"time"

	"github.com/aoscloud/aos_common/aoserrors"
	"github.com/aoscloud/aos_vis/dataprovider"
)

/*******************************************************************************
 * Consts
 ******************************************************************************/

const subscribeWaitTimeout = 100 * time.Millisecond

/*******************************************************************************
 * Types
 ******************************************************************************/

// TestAdapterInfo contains info for adapter test.
type TestAdapterInfo struct {
	Adapter          dataprovider.DataAdapter
	Name             string
	PathListLen      int
	SetData          map[string]interface{}
	SetSubscribeData map[string]interface{}
	SubscribeList    []string
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// GetName tests GetName adapter method.
func GetName(adapterInfo *TestAdapterInfo) (err error) {
	name := adapterInfo.Adapter.GetName()
	if name != adapterInfo.Name {
		return aoserrors.Errorf("wrong adapter %s name: %s", adapterInfo.Name, name)
	}

	return nil
}

// GetPathList tests GetPathList adapter method.
func GetPathList(adapterInfo *TestAdapterInfo) (err error) {
	pathList, err := adapterInfo.Adapter.GetPathList()
	if err != nil {
		return aoserrors.Wrap(err)
	}

	if adapterInfo.PathListLen != 0 && len(pathList) != adapterInfo.PathListLen {
		return aoserrors.Errorf("wrong adapter %s path list len: %d", adapterInfo.Name, len(pathList))
	}

	return nil
}

// PublicPath tests IsPathPublic adapter method.
func PublicPath(adapterInfo *TestAdapterInfo) (err error) {
	pathList, _ := adapterInfo.Adapter.GetPathList()
	for _, path := range pathList {
		_, err := adapterInfo.Adapter.IsPathPublic(path)
		if err != nil {
			return aoserrors.Wrap(err)
		}
	}

	return nil
}

// GetSetData tests Get and Set adapter methods.
func GetSetData(adapterInfo *TestAdapterInfo) (err error) {
	if adapterInfo.SetData == nil {
		return nil
	}

	// set data
	err = adapterInfo.Adapter.SetData(adapterInfo.SetData)
	if err != nil {
		return aoserrors.Wrap(err)
	}

	// get data
	getPathList := make([]string, 0, len(adapterInfo.SetData))
	for path := range adapterInfo.SetData {
		getPathList = append(getPathList, path)
	}

	getData, err := adapterInfo.Adapter.GetData(getPathList)
	if err != nil {
		return aoserrors.Wrap(err)
	}

	// check data
	for path, data := range getData {
		if !reflect.DeepEqual(adapterInfo.SetData[path], data) {
			return aoserrors.Errorf("wrong path: %s value: %v", path, data)
		}
	}

	return nil
}

// SubscribeUnsubscribe tests Subscribe and Unsubscribe adapter methods.
func SubscribeUnsubscribe(adapterInfo *TestAdapterInfo) (err error) {
	if adapterInfo.SetData == nil {
		return nil
	}

	err = adapterInfo.Adapter.SetData(adapterInfo.SetData)
	if err != nil {
		return aoserrors.Wrap(err)
	}

	// subscribe
	if err = adapterInfo.Adapter.Subscribe(adapterInfo.SubscribeList); err != nil {
		return aoserrors.Wrap(err)
	}

	if err = adapterInfo.Adapter.SetData(adapterInfo.SetSubscribeData); err != nil {
		return aoserrors.Wrap(err)
	}

	select {
	case getData := <-adapterInfo.Adapter.GetSubscribeChannel():
		// check data
		for path, data := range getData {
			if !reflect.DeepEqual(adapterInfo.SetSubscribeData[path], data) {
				return aoserrors.Errorf("wrong path: %s value: %v", path, data)
			}
		}
	case <-time.After(subscribeWaitTimeout):
		return aoserrors.Errorf("waiting for adapter %s data timeout", adapterInfo.Name)
	}

	// unsubscribe
	if err = adapterInfo.Adapter.Unsubscribe(adapterInfo.SubscribeList); err != nil {
		return aoserrors.Wrap(err)
	}

	return nil
}
