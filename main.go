package main

import (
	"log"

	gw_files "github.com/generalworksinc/goutil/files"
)

func main() {
	url := `https://smaregi-pos-public.s3.ap-northeast-1.amazonaws.com/transactionDetail/2025/01/04/jv1n791ktuo.csv.gz?AWSAccessKeyId=ASIA5EBGOX5IX4GX3HLT&Expires=1735996639&Signature=JqzLDq1xzk62yJujID%2BxpBAtgC4%3D&X-Amzn-Trace-Id=Root%3D1-677926cd-4a3fa527f668a6dcddb5cec5%3BParent%3D0a5bd715e3b327d1%3BSampled%3D0%3BLineage%3D1%3A45be6e0c%3A0&x-amz-security-token=IQoJb3JpZ2luX2VjECwaDmFwLW5vcnRoZWFzdC0xIkgwRgIhAJNyEZXVnqjkjuOCYBpSJe2zoYd%2FWTQEoYdErXGcYLwhAiEArtbLqUC6vlA8A7p8y%2F7EOnVox8QmOzwuYZhVr13s1Rcq1AMIFRAAGgw5MDIwMjM3OTA0MTciDPhFSZ6SY6ofTUdrWSqxA%2B23YJSL9R2%2BLPIZCkx0EPF6nCIAY6qfYeUfF7VFliLV%2B%2FqF%2ByLdjZJNtqMnpUoYmV5wtyaEx9d15UqAGwUGuPtxVZ8IgbPWKZL8G30uYyfX0t9MztvJHY3hGXThZ1xedmVG1MHSp9RaMnM6cvL39Nnqf88NgLsrUdxVadBQvz5ih%2B%2Bsl7tZFZ%2Bx0N0xie%2FifYI6xRrzlzkYR1BrzOQHWjt07AR4vB5Ar7tYyOIezaZyFzjuHlSLhWx1bgIq%2B9%2FXsWCe0lqVSUefj36aMR91AKUwPZq5mgJkX2nLEEI4wYFNNHgDLTbNWf6MV3BOauaK2QHX%2BbhUquD%2BVFeFGEJrBoxu5Y8ncPrRNs791D%2BfYYk28GOuDsxDAk7LGWvOEQFwNK9T81YwX1pkKadnq9Mh%2B1YT1rgCGOoDgN%2FVxww57Y8LUQMeSGEUFfxyLbGrShtKAvKPwHZzvnsXnu3IH9TtiDaBKse5sNx65xUpHHXW9r8qjge%2BD9H1wUwj%2B0aC37KgBQjDmfQOvBF6U%2B1624ClegrRmm%2B4WA%2BEj5snVNrk7xvqUXeM3frO93hQV2sqmb7M2agwzc3kuwY6nQHUM7CMxoeIQBcvFMrIV%2BHz64nI%2FLRxti58TRGvfYRzXvMVF8MrlDWMlNEGjFg%2BUFDTCBGs4Zd1M8KMIarGMMA6Oqd0UfubonAoEhwxVtpuTonEx77JyL4ap4A51n9Nljamu4d%2F1VsolKDj24q94OAdVSKvEsnEOQdfMzc%2FnwFxDyQmdC9K2CknMeZ6%2BOoH0KGiU7u47x4gfx6H2JE2`
	path, err := gw_files.DownloadFile(url, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("path:", path)
	println("done! ok")
}
