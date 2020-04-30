package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type SupplyChain struct {
}

//
type Orders struct {
	OrderId     string `json:"orderid"`
	TruckId     string `json:"truckid"`
	GpscordsX   string `json:"gpscordsx"`
	GpscordsY   string `json:"gpscordsy"`
}

func constructOrderKey (orderId string) string{
	return fmt.Sprintf("order_%s",orderId)
}

func constructTruckkey (truckId string) string{
	return fmt.Sprintf("truck_%s", truckId)
}

func (c *SupplyChain) Init(stub shim.ChaincodeStubInterface) pb.Response{
	return shim.Success(nil)
}

func (c *SupplyChain) Invoke(stub shim.ChaincodeStubInterface) pb.Response{
	funcName, args := stub.GetFunctionAndParameters()
	switch funcName {
	case "ordersRegister":
		return c.ordersRegister (stub, args)
	case "gpsChange":
		return c.gpsChange (stub, args)
	case "queryOrders":
		return c.queryOrders (stub, args)
	case "queryOrdersHistory":
		return c.queryOrdersHistory (stub, args)
	case "gpsChangeBaseTruck":
		return c.gpsChangeBaseTruck (stub,args)
	default:
		return shim.Error(fmt.Sprintf("unsupported function: %s",funcName))
	}
}

func (c *SupplyChain)ordersRegister(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args)!= 4 {
		return shim.Error("not enough args")
	}
	objectid := args[0]
	truckid := args[1]
	gpscordsx := args[2]
	gpscordsy := args[3]

	if objectid == "" || truckid == "" {
		return shim.Error("invalid args")
	}

	if userBytes, err :=stub.GetState(constructOrderKey(objectid)); err ==nil && len(userBytes) !=0 {
		return shim.Error("order already exist")
	}

	orders := &Orders{
		OrderId:     objectid,
		TruckId:     truckid,
		GpscordsX:   gpscordsx,
		GpscordsY:   gpscordsy,
	}

	userBytes, err := json.Marshal(orders)
	if err != nil{
		return shim.Error(fmt.Sprintf("marshal order error %s",err))
	}
	err = stub.PutState(constructOrderKey(objectid),userBytes)
	if err != nil{
		return shim.Error(fmt.Sprintf("put order error %s",err))
	}
	return shim.Success(nil)
}

func (c *SupplyChain) gpsChange(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3{
		return shim.Error("not enough args")
	}

	objectid := args[0]
	newgpscordsx := args[1]
	newgpscordsy := args[2]
	if objectid == "" || newgpscordsx == "" ||newgpscordsy == ""{
		return shim.Error("invalid args")
	}
	fmt.Println("- start change gps ", objectid, newgpscordsx,newgpscordsy)

	orderBytes ,err := stub.GetState (constructOrderKey(objectid))
	if err !=nil || len(orderBytes) == 0{
		return shim.Error("order not found")
	}

	gpstochange := Orders{}
	err = json.Unmarshal(orderBytes,&gpstochange)
	if err !=nil{
		return shim.Error("Unmarshal not success")
	}
	gpstochange.GpscordsX = newgpscordsx
	gpstochange.GpscordsY = newgpscordsy

	gpsJsonToChange, err := json.Marshal(gpstochange)
	if err != nil{
		return shim.Error(fmt.Sprintf("marshal order error %s",err))
	}
	err = stub.PutState(constructOrderKey(objectid),gpsJsonToChange)
	if err != nil{
		return shim.Error(fmt.Sprintf("put order error %s",err))
	}
	return shim.Success(nil)
}

func (c *SupplyChain) queryOrders(stub shim.ChaincodeStubInterface, args []string) pb.Response{
	if len(args) !=1{
		return shim.Error("not enough args")
	}
	objecrId := args[0]
	if objecrId == ""{
		return shim.Error("invalid args")
	}

	orderBytes ,err := stub.GetState (constructOrderKey(objecrId))
	if err !=nil || len(orderBytes) == 0{
		return shim.Error("order not found")
	}
	return shim.Success(orderBytes)
}

func (c *SupplyChain) gpsChangeBaseTruck(stub shim.ChaincodeStubInterface, args []string) pb.Response{
	//   0          1      2
	// "truckid",   "x"     "y"
	if len(args) !=3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	truckid := args[0]
	newgpscordsx := args[1]
	newgpscordsy := args[2]
	fmt.Println("- start gpsChangeBaseTruck ", truckid, newgpscordsx,newgpscordsy)


	truckidResultsIterator, err := stub.GetStateByPartialCompositeKey("truckid~orderid", []string{truckid})
	if err != nil {
		return shim.Error(err.Error())
	}
	defer truckidResultsIterator.Close()

	// Iterate through result set and for each marble found, transfer to newOwner
	var i int
	for i = 0; truckidResultsIterator.HasNext(); i++ {

		responseRange, err := truckidResultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		// get the color and name from color~name composite key
		objectType, compositeKeyParts, err := stub.SplitCompositeKey(responseRange.Key)
		if err != nil {
			return shim.Error(err.Error())
		}
		returnedtruckid := compositeKeyParts[0]
		returnedorderid := compositeKeyParts[1]
		fmt.Printf("- found a order from index:%s truck:%s order:%s\n", objectType, returnedtruckid, returnedorderid)

		// Now call the transfer function for the found marble.
		// Re-use the same function that is used to transfer individual marbles
		response := c.gpsChange(stub, []string{returnedorderid, newgpscordsx,newgpscordsy})
		// if the transfer failed break out of loop and return error
		if response.Status != shim.OK {
			return shim.Error("change falet: " + response.Message)
		}
	}

	responsePayload := fmt.Sprintf("change %d %s truck to %s %s", i, truckid, newgpscordsx,newgpscordsy)
	fmt.Println("- end gpsChangeBaseTruck: " + responsePayload)
	return shim.Success([]byte(responsePayload))
}
func (c *SupplyChain) queryOrdersHistory(stub shim.ChaincodeStubInterface, args []string) pb.Response{
	if len(args) !=1{
		return shim.Error("not enough args")
	}
	objectID :=args[0]
	if objectID == ""{
		return shim.Error("invalid args")
	}
	resultIterator,err := stub.GetHistoryForKey(constructOrderKey(objectID))
	if err != nil {
		return shim.Error("order not found")
	}
	defer resultIterator.Close()
	var buffer bytes.Buffer
	buffer.WriteString("[")

	isWrited := false
	for resultIterator.HasNext(){
		queryResponse, err :=resultIterator.Next()
		if err != nil{
			return shim.Error(err.Error())
		}
		if isWrited ==true{
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"TxId\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.TxId)
		buffer.WriteString("\"")

		buffer.WriteString(",\"Value\":")
		if queryResponse.IsDelete{
			buffer.WriteString("null")
		}else {
			buffer.WriteString(string(queryResponse.Value))
		}

		buffer.WriteString(",\"Timestamp\":")
		buffer.WriteString("\"")
		buffer.WriteString(time.Unix(queryResponse.Timestamp.Seconds,int64(queryResponse.Timestamp.Nanos)).String())
		buffer.WriteString("\"")

		buffer.WriteString(",\"IsDelete\":")
		buffer.WriteString("\"")
		buffer.WriteString(strconv.FormatBool(queryResponse.IsDelete))
		buffer.WriteString("\"")

		buffer.WriteString("}")
		isWrited = true
	}
	buffer.WriteString("}")
	fmt.Printf("getHistoruForOrders result : \n%s\n", buffer.String())
	fmt.Println("end getHistoryforOrders")

	return shim.Success(buffer.Bytes())
}
func main(){
	err := shim.Start(new(SupplyChain))
	if err != nil {
		fmt.Printf("Error starting SupllyChain chaincode: %s",err)
	}
}
