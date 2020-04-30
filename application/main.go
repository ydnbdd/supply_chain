package main

import (
	"fmt"
	"context"
	"time"
	"net/http"
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/event"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	)

func main(){
	router := gin.Default()
	{
		router.POST("/orders",ordersRegister)
		router.GET("/orders/get/:id",queryOrders)
		router.POST("/orders/change",gpsChange)
		router.GET("/orders/change/history",queryOrdersHistory)
		router.POST("/truck/:id",gpsChangeBaseTruck)
	}
	router.Run()
}

type ordersRegisterRequest struct {
	OrderId     string `from:"orderid" binding:"required"`
	TruckId     string `from:"truckid" binding:"required"`
	GpscordsX   string `from:"gpscordsx" binding:"required"`
	GpscordsY   string `from:"gpscordsy" binding:"required"`
}

func ordersRegister(ctx *gin.Context) {
	req := new(ordersRegisterRequest)
	if err := ctx.ShouldBind(req);err !=nil{
		ctx.AbortWithError(400,err)
		return
	}

	resp, err := channelExecute("ordersRegister",[][]byte{
		[]byte(req.OrderId),
		[]byte(req.TruckId),
		[]byte(req.GpscordsX),
		[]byte(req.GpscordsY),
	})
	if err !=nil{
		ctx.String(http.StatusOK,err.Error())
	}
	ctx.JSON(http.StatusOK,resp)
}

func queryOrders(ctx *gin.Context) {
	orderid := ctx.Param("id")
	resp, err := channelQuery("queryOrders",[][]byte{
		[]byte(orderid),
	})
	if err != nil{
		ctx.String(http.StatusOK,err.Error())
		return
	}
	ctx.String(http.StatusOK,bytes.NewBuffer(resp.Payload).String())
}

type gpsChangeRequest struct {
	OrderId     string `from:"orderid" binding:"required"`
	GpscordsX   string `from:"gpscordsx" binding:"required"`
	GpscordsY   string `from:"gpscordsy" binding:"required"`
}
func gpsChange(ctx *gin.Context) {
	req := new(gpsChangeRequest)
	if err := ctx.ShouldBind(req); err != nil{
		ctx.AbortWithError(400,err)
		return
	}
	resp, err := channelExecute("gpsChange",[][]byte{
		[]byte(req.OrderId),
		[]byte(req.GpscordsX),
		[]byte(req.GpscordsY),
	})
	if err != nil{
		ctx.String(http.StatusOK,err.Error())
		return
	}
	ctx.JSON(http.StatusOK,resp)
}

func queryOrdersHistory(ctx *gin.Context) {
	orderId := ctx.Param("id")
	resp, err := channelQuery("queryOrdersHistory",[][]byte{
		[]byte(orderId),
	})
	if err != nil{
		ctx.String(http.StatusOK,err.Error())
		return
	}
	ctx.String(http.StatusOK,bytes.NewBuffer(resp.Payload).String())
}

type gpsChangeBaseTruckRequest struct {
	TruckId     string `from:"truckid" binding:"required"`
	GpscordsX   string `from:"gpscordsx" binding:"required"`
	GpscordsY   string `from:"gpscordsy" binding:"required"`
}
func gpsChangeBaseTruck(ctx *gin.Context) {
	req := new(gpsChangeBaseTruckRequest)
	if err := ctx.ShouldBind(req);err != nil{
		ctx.AbortWithError(400,err)
		return
	}
	resp, err := channelExecute("gpsChangeBaseTruck",[][]byte{
		[]byte(req.TruckId),
		[]byte(req.GpscordsX),
		[]byte(req.GpscordsY),
	})
	if err !=nil {
		ctx.String(http.StatusOK,err.Error())
		return
	}
	ctx.JSON(http.StatusOK,resp)
}

var (
	sdk           *fabsdk.FabricSDK
	channelName   = "mychannel"
	chaincodeName = "mycc1"
	org           = "org1"
	user          = "Admin"
	configPath    = "/home/ydnbdd/go/src/github.com/hyperledger/fabric/examples/application/config.yaml"
)

func init(){
	var err error
	sdk, err =fabsdk.New(config.FromFile(configPath))
	if err !=nil{
		panic(err)
	}
}

func manageBlockchain(){
	ctx := sdk.Context(fabsdk.WithOrg(org),fabsdk.WithUser(user))
	cli, err :=resmgmt.New(ctx)
	if err != nil{
		panic(err)
	}
	cli.SaveChannel(resmgmt.SaveChannelRequest{},resmgmt.WithOrdererEndpoint("orderer.example.com"), resmgmt.WithTargetEndpoints())
}

func queryBlockchain(){
	ctx := sdk.ChannelContext(channelName,fabsdk.WithOrg(org),fabsdk.WithUser(user))
	cli, err := ledger.New(ctx)
	if err !=nil {
		panic(err)
	}
	resp, err := cli.QueryInfo(ledger.WithTargetEndpoints("peer0.org1.example.com"))
	if err !=nil{
		panic(err)
	}
	fmt.Println(resp)
	for i := uint64(0);i<=resp.BCI.Height ;i++  {
		cli.QueryBlock(i)
	}
}

func channelExecute(fcn string,args[][]byte)(channel.Response,error){
	ctx := sdk.ChannelContext(channelName,fabsdk.WithOrg(org),fabsdk.WithUser(user))

	cli, err := channel.New(ctx)
	if err !=nil {
		return channel.Response{},err
	}

	resp ,err :=cli.Execute(channel.Request{
		ChaincodeID:     chaincodeName,
		Fcn:             fcn,
		Args:            args,
	},channel.WithTargetEndpoints("peer0.org1.example.com"))
	if err != nil{
		return channel.Response{},err
	}
	go func() {
		reg, ccevt,err := cli.RegisterChaincodeEvent(chaincodeName,"eventname")
		if err != nil{
			return
		}
		defer cli.UnregisterChaincodeEvent(reg)

		timeoutctx, cancel := context.WithTimeout(context.Background(),time.Minute)
		defer cancel()
		for{
			select {
			case evt := <-ccevt:
				fmt.Printf("received event of tx %s : %+v",resp.TransactionID,evt)
			case <-timeoutctx.Done():
				fmt.Println("event timeout,exit!")
				return
			}
		}
	}()

	go func() {
		eventcli, err := event.New(ctx)
		if err != nil{
			return
		}
		reg,status,err := eventcli.RegisterTxStatusEvent(string(resp.TransactionID))
		defer eventcli.Unregister(reg)

		timeoutctx, cancel := context.WithTimeout(context.Background(),time.Minute)
		defer cancel()
		for  {
			select {
			case evt := <-status :
				fmt.Printf("received event of tx %s: %+v", resp.TransactionID, evt)
			case <-timeoutctx.Done():
				fmt.Println("event timeout, exit!")
				return
			}
		}

		
	}()
	return resp,nil
}

func channelQuery(fcn string, args [][]byte) (channel.Response, error)  {
	ctx := sdk.ChannelContext(channelName, fabsdk.WithOrg(org), fabsdk.WithUser(user))

	cli, err := channel.New(ctx)
	if err != nil {
		return channel.Response{}, err
	}
	return cli.Query(channel.Request{
		ChaincodeID:     chaincodeName,
		Fcn:             fcn,
		Args:            args,
	},channel.WithTargetEndpoints("peer0.org1.example.com"))
}

func eventHandle(){
	ctx := sdk.ChannelContext(channelName, fabsdk.WithOrg(org),fabsdk.WithUser(user))
	cli, err := event.New(ctx)
	if err != nil {
		panic(err)
	}
	reg, blkevent,err :=cli.RegisterBlockEvent()
	if err!=nil {
		panic(err)
	}
	defer cli.Unregister(reg)
	timeoutctx, cancel := context.WithTimeout(context.Background(),time.Minute)
	defer cancel()
	for  {
		select {
		case evt := <-blkevent:
			fmt.Printf("received a block", evt)
		case <-timeoutctx.Done():
			fmt.Println("event timeout, exit!")
			return
			
		}
	}

}