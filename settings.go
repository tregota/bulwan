package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"

	"github.com/iancoleman/strcase"
)

// LoadSettings fills settings object provided via a pointer by looking in files and env vars
func LoadSettings(settings interface{}) error {
	// first and with lowest priority, look for a settings.conf file
	if _, err := os.Stat("settings.conf"); !os.IsNotExist(err) {
		file, err := os.Open("settings.conf")
		if err != nil {
			return err
		}
		decoder := json.NewDecoder(file)
		err = decoder.Decode(settings)
		if err != nil {
			return err
		}
	}
	// ValueOf makes a Value instance with copied contents from the provided pointer and Elem() gets the Value that the pointer points to, ie the settings struct
	reflectValue := reflect.ValueOf(settings).Elem()
	reflectType := reflectValue.Type()
	for i := 0; i < reflectValue.NumField(); i++ {
		valueField := reflectValue.Field(i)
		typeField := reflectType.Field(i)
		settingName := typeField.Name
		settingType := typeField.Type
		var settingValue string

		// find setting
		if typeField.Type.Kind() == reflect.Slice {
			innerReflectType := typeField.Type.Elem()
			tempSlice := make([]reflect.Value, 0) // temp slice since I cannot figure out how to use reflect.Append here

		INNEROBJECTS:
			for {
				innerReflectValue := reflect.New(innerReflectType).Elem()
				for j := 0; j < innerReflectValue.NumField(); j++ {
					innerValueField := innerReflectValue.Field(j)
					innerTypeField := innerReflectType.Field(j)
					fullname := fmt.Sprintf("%s%s%d", settingName, innerTypeField.Name, len(tempSlice)+1)

					if innerValue, ok := os.LookupEnv(strcase.ToScreamingSnake(fullname)); !ok {
						break INNEROBJECTS
					} else {
						result, err := getValue(innerTypeField.Type, innerValue)
						if err == nil {
							innerValueField.Set(result)
						}
					}
				}
				tempSlice = append(tempSlice, innerReflectValue)
			}

			if len(tempSlice) > 0 {
				newSlice := reflect.MakeSlice(settingType, len(tempSlice), len(tempSlice))
				for idx, value := range tempSlice {
					newSlice.Index(idx).Set(value)
				}
				valueField.Set(newSlice)
			}

		} else {
			if result, ok := os.LookupEnv(strcase.ToScreamingSnake(settingName)); ok {
				settingValue = result
			} else if _, err := os.Stat(settingName); !os.IsNotExist(err) {
				result, err := ioutil.ReadFile(settingName)
				if err != nil {
					return err
				} else {
					settingValue = string(result) // convert content to string
				}
			}
		}

		// set setting
		result, err := getValue(settingType, settingValue)
		if err == nil {
			valueField.Set(result)
		}
	}
	return nil
}

func getValue(settingType reflect.Type, settingValue string) (reflect.Value, error) {
	if len(settingValue) > 0 {
		switch settingType.Name() {
		case "string":
			return reflect.ValueOf(settingValue), nil
		case "int":
			intvalue, err := strconv.Atoi(settingValue)
			if err != nil {
				return reflect.ValueOf(nil), err
			}
			return reflect.ValueOf(intvalue), nil
		default:
			return reflect.ValueOf(settingValue), nil
		}
	}
	return reflect.ValueOf(nil), errors.New("getValue - no value")
}
