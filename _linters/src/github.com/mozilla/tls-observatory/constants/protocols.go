package constants

type Protocol struct {
	OpenSSLName string `json:"openssl_name"`
	Code        int    `json:"code"`
}

var Protocols = []Protocol{
	Protocol{
		OpenSSLName: "SSLv3",
		Code:        768,
	},
	Protocol{
		OpenSSLName: "TLSv1",
		Code:        769,
	},
	Protocol{
		OpenSSLName: "TLSv1.1",
		Code:        770,
	},
	Protocol{
		OpenSSLName: "TLSv1.2",
		Code:        771,
	},
}
