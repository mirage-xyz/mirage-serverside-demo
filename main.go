package main

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"math/big"
	"net/http"

	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/gin-gonic/gin"
)

// Endpoint for evm rpc requests
var infura = "https://rinkeby.infura.io/v3/c75f2ce78a4a4b64aa1e9c20316fda3e"
var client, clientConnectErr = ethclient.Dial(infura)

// A simple ERC-20 token on the testnet
var contractAccount = common.HexToAddress("0xb48366c616c7Ce992981cFB354301Da161687855")

// Private key on the server side For GD-3 (use case 8)
var privateKey, _ = crypto.HexToECDSA("6e97855fb478f18012146750022a417cb46dddc9814f6c46a22b34b71a2d0074")

// user's address associated with the hero id
var clientAddress = common.HexToAddress("0x24d13b65bAbFc38f6eCA86D9e73C539a1e0C0196")

var typesStandard = apitypes.Types{
	"EIP712Domain": {
		{
			Name: "name",
			Type: "string",
		},
		{
			Name: "version",
			Type: "string",
		},
		{
			Name: "chainId",
			Type: "uint256",
		},
		{
			Name: "verifyingContract",
			Type: "address",
		},
	},
	"ItemInfo": {
		{
			Name: "tokenId",
			Type: "uint256",
		},
		{
			Name: "itemType",
			Type: "uint256",
		},
		{
			Name: "strength",
			Type: "uint256",
		},
		{
			Name: "level",
			Type: "uint256",
		},
		{
			Name: "expireTime",
			Type: "uint256",
		},
	},
}

const primaryType = "ItemInfo"

var domainStandard = apitypes.TypedDataDomain{
	Name:              "GameItem",
	Version:           "1",
	ChainId:           math.NewHexOrDecimal256(4),
	VerifyingContract: contractAccount.String(),
}

type Item struct {
	itemType int `json: itemType`
	strength int `json: strength`
	level    int `json: level`
}

type ItemInfo struct {
	tokenId    string `json: tokenId`
	itemType   int    `json: itemType`
	strength   int    `json: strength`
	level      int    `json: level`
	expireTime int    `json: expireTime`
	signature  string `json: signature`
}

type Payload struct {
	hero      ItemInfo `json: hero`
	signature string   `json: signature`
}

type SignatureCheckInput struct {
	Signature string `json: "signature"`
	Message   string `json: "message"`
}

type SignatureCheckOutput struct {
	Address string `json: "address"`
}

func main() {
	r := setupRouter()
	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}

func setupRouter() *gin.Engine {
	// Disable Console Color
	// gin.DisableConsoleColor()
	r := gin.Default()

	// GD-3: Fetch the hero data based on its ID and return the transaction to be signed.
	r.GET("/hero/:id", func(c *gin.Context) {
		// in this example, we don't have backend database, the desired actions should be
		// 1) check if the id exists in db
		// 2) fetch the clientAddress associate with this id
		// 3) prepare the transaction to be signed by the users to update the properties of NFT heros.
		id := c.Params.ByName("id")

		if clientConnectErr != nil {
			log.Fatal(clientConnectErr)
		}

		// parsedAbi, _ := abi.JSON(strings.NewReader(ERC20.ERC20ABI))
		// bytesData, _ := parsedAbi.Pack("mint")
		// nonce, _ := client.NonceAt(context.Background(), clientAddress, nil)
		// tx := types.NewTransaction(nonce, clientAddress, nil, big.NewInt(10000000).Uint64(), big.NewInt(0), bytesData)
		// signedTx, _ := types.SignTx(tx, types.HomesteadSigner{}, privateKey)

		// messageStandard := map[string]interface{}{
		// 	"tokenId":    "",
		// 	"itemType":   "",
		// 	"strength":   "",
		// 	"level":      "",
		// 	"expireTime": "",
		// }

		// typedData := apitypes.TypedData{
		// 	Types:       typesStandard,
		// 	PrimaryType: primaryType,
		// 	Domain:      domainStandard,
		// 	Message:     messageStandard,
		// }

		// ExternalAPI.SignTypedData(context.Background(), a, typedData)

		hero := &ItemInfo{
			tokenId:  id,
			itemType: 1,
			strength: 10,
			level:    15,
		}

		// r, s, v := signedTx.RawSignatureValues()
		// payload := &Payload{hero: *hero, signature: signatureToHex(r, s, v)}

		// payloadStr, err := json.Marshal(payload)
		payloadStr, err := json.Marshal(hero)
		if err == nil {
			c.Data(http.StatusOK, gin.MIMEJSON, payloadStr)
		}
	})

	// GD-2: Verify message and signature, if passed, bind this address to user's table
	// Input: message and signature
	// Output: Users' address
	r.POST("/account/verification/address", func(c *gin.Context) {
		var input SignatureCheckInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		data := []byte(input.Message)

		// Verify the Signature and message, if passed, return pub key
		sigPublicKey := getAddrFromSign(input.Signature, data)
		// Convert pub key to address
		address := string(sigPublicKey)

		// add address to a database
		output := &SignatureCheckOutput{Address: address}

		// return address
		payloadStr, err := json.Marshal(output)
		if err == nil {
			c.Data(http.StatusOK, gin.MIMEJSON, payloadStr)
		}
	})

	return r
}

func signatureToHex(r *big.Int, s *big.Int, v *big.Int) string {
	return "0x" + hex.EncodeToString(r.Bytes()) + hex.EncodeToString(s.Bytes()) + hex.EncodeToString(v.Bytes())
}

func getAddrFromSign(sigHex string, msg []byte) string {
	sig := hexutil.MustDecode(sigHex)
	if sig[64] != 27 && sig[64] != 28 {
		log.Fatal("Problem 1")
	}
	sig[64] -= 27

	pubKey, err := crypto.SigToPub(signHash(msg), sig)
	if err != nil {
		log.Fatal(err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	return recoveredAddr.String()
}

func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}

func signTypedData() {

}
