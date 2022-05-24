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

package subjectsadapter_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/aoscloud/aos_common/aoserrors"
	"github.com/aoscloud/aos_common/api/visprotocol"
	subjectsadapter "github.com/aoscloud/aos_vis/plugins/subjectsadapter"
	log "github.com/sirupsen/logrus"
)

/*******************************************************************************
 * Consts
 ******************************************************************************/

const subjectsVISPath = "Attribute.Aos.Subjects"

/*******************************************************************************
 * Vars
 ******************************************************************************/

var tmpDir string

/*******************************************************************************
 * Init
 ******************************************************************************/

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		TimestampFormat:  "2006-01-02 15:04:05.000",
		FullTimestamp:    true,
	})
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
}

/*******************************************************************************
 * Main
 ******************************************************************************/

func TestMain(m *testing.M) {
	var err error

	tmpDir, err = ioutil.TempDir("", "vis_")
	if err != nil {
		log.Fatalf("Error creating tmp dir: %s", err)
	}

	ret := m.Run()

	if err := os.RemoveAll(tmpDir); err != nil {
		log.Fatalf("Error removing tmp dir: %s", err)
	}

	os.Exit(ret)
}

/*******************************************************************************
 * Tests
 ******************************************************************************/

func TestGetName(t *testing.T) {
	adapter, err := subjectsadapter.New(generateConfig(path.Join(tmpDir, "subject.txt")))
	if err != nil {
		t.Fatalf("Can't create adapter: %s", err)
	}
	defer adapter.Close()

	if name := adapter.GetName(); name != "subjectsadapter" {
		t.Errorf("Wrong adapter name: %s", name)
	}
}

func TestEmptysubject(t *testing.T) {
	subjectFile := path.Join(tmpDir, "subjects.txt")
	if err := os.RemoveAll(subjectFile); err != nil {
		t.Fatalf("Can't remove Subjects file: %s", err)
	}

	adapter, err := subjectsadapter.New(generateConfig(subjectFile))
	if err != nil {
		t.Fatalf("Can't create adapter: %s", err)
	}
	defer adapter.Close()

	data, err := adapter.GetData([]string{subjectsVISPath})
	if err != nil {
		t.Fatalf("Can't get data: %s", err)
	}

	if _, ok := data[subjectsVISPath]; !ok {
		t.Fatal("Subject not found in data")
	}

	subjects, ok := data[subjectsVISPath].([]string)
	if !ok {
		t.Fatal("Wrong subjects data type")
	}

	if !reflect.DeepEqual(subjects, []string{}) {
		t.Errorf("Wrong subjects value: %s", subjects)
	}
}

func TestExistingSubject(t *testing.T) {
	subjectFile := path.Join(tmpDir, "subjects.txt")
	originSubjects := []string{"claim0", "claim1", "claim2"}

	if err := writeSubjects(subjectFile, originSubjects); err != nil {
		t.Fatalf("Can't create subjects file: %s", err)
	}

	adapter, err := subjectsadapter.New(generateConfig(subjectFile))
	if err != nil {
		t.Fatalf("Can't create adapter: %s", err)
	}
	defer adapter.Close()

	data, err := adapter.GetData([]string{subjectsVISPath})
	if err != nil {
		t.Fatalf("Can't get data: %s", err)
	}

	if _, ok := data[subjectsVISPath]; !ok {
		t.Fatal("Subjects not found in data")
	}

	subjects, ok := data[subjectsVISPath].([]string)
	if !ok {
		t.Fatal("Wrong subjects data type")
	}

	if !reflect.DeepEqual(originSubjects, subjects) {
		t.Errorf("Wrong Subjects value: %s", subjects)
	}
}

func TestSetSubject(t *testing.T) {
	subjectsFile := path.Join(tmpDir, "subjects.txt")
	if err := os.RemoveAll(subjectsFile); err != nil {
		t.Fatalf("Can't remove subjects file: %s", err)
	}

	adapter, err := subjectsadapter.New(generateConfig(subjectsFile))
	if err != nil {
		t.Fatalf("Can't create adapter: %s", err)
	}
	defer adapter.Close()

	setSubjectsTestSet := [][]string{
		{"claim0", "claim1", "claim2"},
		{"claim3"},
		{},
	}

	if err = adapter.Subscribe([]string{subjectsVISPath}); err != nil {
		t.Fatalf("Subscribe error: %s", err)
	}

	for setIndex := range setSubjectsTestSet {
		setSubjects := make([]interface{}, len(setSubjectsTestSet[setIndex]))
		for i, v := range setSubjectsTestSet[setIndex] {
			setSubjects[i] = v
		}

		if err = adapter.SetData(map[string]interface{}{subjectsVISPath: setSubjects}); err != nil {
			t.Fatalf("Set data error: %s", err)
		}

		select {
		case data := <-adapter.GetSubscribeChannel():
			if !reflect.DeepEqual(data[subjectsVISPath], setSubjects) {
				t.Errorf("Wrong subjects value: %s", setSubjects)
			}

		case <-time.After(5 * time.Second):
			t.Error("Wait data change timeout")
		}
	}
}

func TestSetSubjectFromJson(t *testing.T) {
	subjectsFile := path.Join(tmpDir, "subjects.txt")
	if err := os.RemoveAll(subjectsFile); err != nil {
		t.Fatalf("Can't remove subjects file: %s", err)
	}

	adapter, err := subjectsadapter.New(generateConfig(subjectsFile))
	if err != nil {
		t.Fatalf("Can't create adapter: %s", err)
	}

	defer adapter.Close()

	setRequest := `{
		"action": "set",
		"requestId": "d1d735bf-40ae-4ac3-a68c-d1d60368c83b",
		"path": "Attribute.Aos.Subjects",
		"value": ["428efde9-76e7-4532-9024-50b6b292fea6"]
	}`

	request := visprotocol.SetRequest{}

	if err := json.Unmarshal([]byte(setRequest), &request); err != nil {
		t.Fatalf("Can't unmarshall request: %s", err)
	}

	if err := adapter.SetData(map[string]interface{}{subjectsVISPath: request.Value}); err != nil {
		t.Fatalf("Can't set data: %s", err)
	}

	data, err := adapter.GetData([]string{subjectsVISPath})
	if err != nil {
		t.Fatalf("Can't get data: %s", err)
	}

	_, ok := data[subjectsVISPath]
	if !ok {
		t.Fatal("Subjects not found in data")
	}

	subjects, ok := data[subjectsVISPath].([]string)
	if !ok {
		t.Fatal("Wrong subjects data type")
	}

	if len(subjects) != 1 {
		t.Errorf("Wrong count of subjects")
	}

	if subjects[0] != "428efde9-76e7-4532-9024-50b6b292fea6" {
		t.Errorf("Wrong value of subjects")
	}
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func generateConfig(filePath string) (config []byte) {
	type adapterConfig struct {
		VISPath  string `json:"visPath"`
		FilePath string `json:"filePath"`
	}

	var err error

	if config, err = json.Marshal(&adapterConfig{VISPath: subjectsVISPath, FilePath: filePath}); err != nil {
		log.Fatalf("Can't marshal config: %s", err)
	}

	return config
}

func writeSubjects(subjectsFile string, subjects []string) (err error) {
	file, err := os.Create(subjectsFile)
	if err != nil {
		return aoserrors.Wrap(err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, claim := range subjects {
		fmt.Fprintln(writer, claim)
	}

	return aoserrors.Wrap(writer.Flush())
}
