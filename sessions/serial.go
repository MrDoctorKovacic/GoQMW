package sessions

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MrDoctorKovacic/MDroid-Core/logging"
	"github.com/MrDoctorKovacic/MDroid-Core/mserial"
	"github.com/tarm/serial"
)

// ReadFromSerial reads serial data into the session
func ReadFromSerial(device *serial.Port) bool {
	status.Log(logging.OK(), "Starting serial read")
	for connected := true; connected; {
		response, err := mserial.ReadSerial(device)
		// The device is nil, break out of this read loop
		if err != nil {
			status.Log(logging.Error(), err.Error())
			break
		}
		parseSerialJSON(response)
	}
	return true
}

func parseSerialJSON(marshalledJSON interface{}) {
	if marshalledJSON == nil {
		status.Log(logging.Error(), " marshalled JSON is nil.")
		return
	}

	data := marshalledJSON.(map[string]interface{})

	// Switch through various types of JSON data
	for key, value := range data {
		switch vv := value.(type) {
		case bool:
			SetValue(strings.ToUpper(key), strings.ToUpper(strconv.FormatBool(vv)))
		case string:
			SetValue(strings.ToUpper(key), strings.ToUpper(vv))
		case int:
			SetValue(strings.ToUpper(key), strconv.Itoa(value.(int)))
		case float32:
			floatValue, ok := value.(float32)
			if ok {
				SetValue(strings.ToUpper(key), fmt.Sprintf("%f", floatValue))
			}
		case float64:
			floatValue, ok := value.(float64)
			if ok {
				SetValue(strings.ToUpper(key), fmt.Sprintf("%f", floatValue))
			}
		case []interface{}:
			status.Log(logging.Error(), key+" is an array. Data: ")
			for i, u := range vv {
				fmt.Println(i, u)
			}
		case nil:
			break
		default:
			status.Log(logging.Error(), fmt.Sprintf("%s is of a type I don't know how to handle (%s: %s)", key, vv, value))
		}
	}
}
