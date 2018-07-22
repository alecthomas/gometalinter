package constants

// Curve is the definition of an elliptic curve
type Curve struct {
	Name        string `json:"iana_name"`
	OpenSSLName string `json:"openssl_name,omitempty"`
	PFSName     string `json:"pfs_name,omitempty"`
	Code        uint64 `json:"code"`
}

// Curves is a list of known IANA curves with their code point,
// IANA name, openssl name and PFS alias used by openssl
var Curves = []Curve{
	Curve{
		Code:        1,
		Name:        "sect163k1",
		OpenSSLName: "",
		PFSName:     "K-163",
	},
	Curve{
		Code:        2,
		Name:        "sect163r1",
		OpenSSLName: "",
		PFSName:     "",
	},
	Curve{
		Code:        3,
		Name:        "sect163r2",
		OpenSSLName: "",
		PFSName:     "B-163",
	},
	Curve{
		Code:        4,
		Name:        "sect193r1",
		OpenSSLName: "",
		PFSName:     "",
	},
	Curve{
		Code:        5,
		Name:        "sect193r2",
		OpenSSLName: "",
		PFSName:     "",
	},
	Curve{
		Code:        6,
		Name:        "sect233k1",
		OpenSSLName: "",
		PFSName:     "K-233",
	},
	Curve{
		Code:        7,
		Name:        "sect233r1",
		OpenSSLName: "",
		PFSName:     "",
	},
	Curve{
		Code:        8,
		Name:        "sect239k1",
		OpenSSLName: "",
		PFSName:     "",
	},
	Curve{
		Code:        9,
		Name:        "sect283k1",
		OpenSSLName: "",
		PFSName:     "K-283",
	},
	Curve{
		Code:        10,
		Name:        "sect283r1",
		OpenSSLName: "",
		PFSName:     "B-283",
	},
	Curve{
		Code:        11,
		Name:        "sect409k1",
		OpenSSLName: "",
		PFSName:     "K-409",
	},
	Curve{
		Code:        12,
		Name:        "sect409r1",
		OpenSSLName: "",
		PFSName:     "B-409",
	},
	Curve{
		Code:        13,
		Name:        "sect571k1",
		OpenSSLName: "",
		PFSName:     "K-571",
	},
	Curve{
		Code:        14,
		Name:        "sect571r1",
		OpenSSLName: "",
		PFSName:     "B-571",
	},
	Curve{
		Code:        15,
		Name:        "secp160k1",
		OpenSSLName: "",
		PFSName:     "",
	},
	Curve{
		Code:        16,
		Name:        "secp160r1",
		OpenSSLName: "",
		PFSName:     "",
	},
	Curve{
		Code:        17,
		Name:        "secp160r2",
		OpenSSLName: "",
		PFSName:     "",
	},
	Curve{
		Code:        18,
		Name:        "secp192k1",
		OpenSSLName: "",
		PFSName:     "",
	},
	Curve{
		Code:        19,
		Name:        "secp192r1",
		OpenSSLName: "prime192v1",
		PFSName:     "P-192",
	},
	Curve{
		Code:        20,
		Name:        "secp224k1",
		OpenSSLName: "",
		PFSName:     "",
	},
	Curve{
		Code:        21,
		Name:        "secp224r1",
		OpenSSLName: "",
		PFSName:     "P-224",
	},
	Curve{
		Code:        22,
		Name:        "secp256k1",
		OpenSSLName: "",
		PFSName:     "",
	},
	Curve{
		Code:        23,
		Name:        "secp256r1",
		OpenSSLName: "prime256v1",
		PFSName:     "P-256",
	},
	Curve{
		Code:        24,
		Name:        "secp384r1",
		OpenSSLName: "",
		PFSName:     "P-384",
	},
	Curve{
		Code:        25,
		Name:        "secp521r1",
		OpenSSLName: "",
		PFSName:     "P-521",
	},
	Curve{
		Code:        26,
		Name:        "brainpoolP256r1",
		OpenSSLName: "",
		PFSName:     "",
	},
	Curve{
		Code:        27,
		Name:        "brainpoolP384r1",
		OpenSSLName: "",
		PFSName:     "",
	},
	Curve{
		Code:        28,
		Name:        "brainpoolP512r1",
		OpenSSLName: "",
		PFSName:     "",
	},
	Curve{
		Code:        29,
		Name:        "ecdh_x25519",
		OpenSSLName: "",
		PFSName:     "",
	},
	Curve{
		Code:        30,
		Name:        "ecdh_x448",
		OpenSSLName: "",
		PFSName:     "",
	},
}
