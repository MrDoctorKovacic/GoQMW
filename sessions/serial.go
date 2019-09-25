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
func (session *Session) ReadFromSerial(device *serial.Port) {
	SessionStatus.Log(logging.OK(), "Starting serial read")
	for connected := true; connected; {
		session.parseSerialJSON(mserial.ReadSerial(device))
	}
}

func (session *Session) parseSerialJSON(marshalledJSON interface{}) {

	if marshalledJSON == nil {
		SessionStatus.Log(logging.Error(), " marshalled JSON is nil.")
		return
	}

	data := marshalledJSON.(map[string]interface{})

	// Switch through various types of JSON data
	for key, value := range data {
		switch vv := value.(type) {
		case bool:
			session.CreateSessionValue(strings.ToUpper(key), strings.ToUpper(strconv.FormatBool(vv)))
		case string:
			session.CreateSessionValue(strings.ToUpper(key), strings.ToUpper(vv))
		case int:
			session.CreateSessionValue(strings.ToUpper(key), strconv.Itoa(value.(int)))
		case float32:
			floatValue, _ := value.(float32)
			session.CreateSessionValue(strings.ToUpper(key), fmt.Sprintf("%f", floatValue))
		case float64:
			floatValue, _ := value.(float64)
			session.CreateSessionValue(strings.ToUpper(key), fmt.Sprintf("%f", floatValue))
		case []interface{}:
			SessionStatus.Log(logging.Error(), key+" is an array. Data: ")
			for i, u := range vv {
				fmt.Println(i, u)
			}
		case nil:
			break
		default:
			SessionStatus.Log(logging.Error(), fmt.Sprintf("%s is of a type I don't know how to handle (%s: %s)", key, vv, value))
		}
	}
}
