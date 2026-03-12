package annualtaxxml

import "encoding/xml"

// OSVC is the root element for CSSZ OSVC annual overview XML.
type OSVC struct {
	XMLName xml.Name     `xml:"OSVC"`
	Xmlns   string       `xml:"xmlns,attr"`
	Version string       `xml:"version,attr"`
	Vendor  Vendor       `xml:"VENDOR"`
	Sender  Sender       `xml:"SENDER"`
	Prehled PrehledOSVC  `xml:"prehledosvc"`
}

// Vendor identifies the software generating the XML.
type Vendor struct {
	ProductName    string `xml:"productName,attr"`
	ProductVersion string `xml:"productVersion,attr"`
}

// Sender contains notification and protocol settings.
type Sender struct {
	EmailNotifikace string `xml:"EmailNotifikace,attr"`
	ISDSreport      string `xml:"ISDSreport,attr"`
	VerzeProtokolu  string `xml:"VerzeProtokolu,attr"`
}

// PrehledOSVC is the main overview element.
type PrehledOSVC struct {
	For  string `xml:"for,attr"`
	Dep  string `xml:"dep,attr"`
	Rok  string `xml:"rok,attr"`
	Typ  string `xml:"typ,attr"`
	VSDP string `xml:"vsdp,attr"`
	Dat  string `xml:"dat,attr"`

	Client Client `xml:"client"`
	PVV    PVV    `xml:"pvv"`
	Prihldp string `xml:"prihldp"`
	Zal    Zal    `xml:"zal"`
	Pre    Pre    `xml:"pre"`
	Prizn  Prizn  `xml:"prizn"`
	Spo    Spo    `xml:"spo"`
	DatEl  DatEl  `xml:"dat"`
}

// Client contains taxpayer personal information.
type Client struct {
	Name  ClientName  `xml:"name"`
	Birth ClientBirth `xml:"birth"`
	Adr   Address     `xml:"adr"`
	IDDS  string      `xml:"idds"`
	Email string      `xml:"email"`
	Tel   string      `xml:"tel"`
	Druc  string      `xml:"druc"`
	Hlavc Hlavc       `xml:"hlavc"`
	Vedc  Vedc        `xml:"vedc"`
	Narok MonthFlags  `xml:"narok"`
	Sleva MonthFlags  `xml:"sleva"`
}

// ClientName holds first/last name and title.
type ClientName struct {
	Fir string `xml:"fir,attr"`
	Sur string `xml:"sur,attr"`
	Tit string `xml:"tit,attr"`
}

// ClientBirth holds birth number and date.
type ClientBirth struct {
	Bno string `xml:"bno,attr"`
	Den string `xml:"den,attr"`
}

// Address holds a Czech postal address.
type Address struct {
	Cit string `xml:"cit,attr"`
	Cnt string `xml:"cnt,attr"`
	Num string `xml:"num,attr"`
	Pnu string `xml:"pnu,attr"`
	Str string `xml:"str,attr"`
}

// MonthFlags represents 13 monthly flag fields (m1-m13).
type MonthFlags struct {
	M1  string `xml:"m1"`
	M2  string `xml:"m2"`
	M3  string `xml:"m3"`
	M4  string `xml:"m4"`
	M5  string `xml:"m5"`
	M6  string `xml:"m6"`
	M7  string `xml:"m7"`
	M8  string `xml:"m8"`
	M9  string `xml:"m9"`
	M10 string `xml:"m10"`
	M11 string `xml:"m11"`
	M12 string `xml:"m12"`
	M13 string `xml:"m13"`
}

// Hlavc contains main activity month flags.
type Hlavc struct {
	MonthFlags
}

// Vedc contains secondary activity month flags and additional fields.
type Vedc struct {
	M1     string `xml:"m1"`
	M2     string `xml:"m2"`
	M3     string `xml:"m3"`
	M4     string `xml:"m4"`
	M5     string `xml:"m5"`
	M6     string `xml:"m6"`
	M7     string `xml:"m7"`
	M8     string `xml:"m8"`
	M9     string `xml:"m9"`
	M10    string `xml:"m10"`
	M11    string `xml:"m11"`
	M12    string `xml:"m12"`
	M13    string `xml:"m13"`
	Zam    string `xml:"zam"`
	Duchod string `xml:"duchod"`
	Pdite  string `xml:"pdite"`
	PPM    string `xml:"ppm"`
	Pece   string `xml:"pece"`
	Ndite  string `xml:"ndite"`
}

// PVV contains income/expense overview data.
type PVV struct {
	Pri string `xml:"pri,attr"`

	Mesc    HVPair `xml:"mesc"`
	Mesv    HVPair `xml:"mesv"`
	Mesp    string `xml:"mesp"`
	Rdza    HVPair `xml:"rdza"`
	VVZ     HVPair `xml:"vvz"`
	DVZ     HVPair `xml:"dvz"`
	MVZ     string `xml:"mvz"`
	UVZ     string `xml:"uvz"`
	Vzza    string `xml:"vzza"`
	Vzsu    string `xml:"vzsu"`
	Vzsvc   string `xml:"vzsvc"`
	Poj     string `xml:"poj"`
	Slev    string `xml:"slev"`
	Pojposlev string `xml:"pojposlev"`
	Zal     string `xml:"zal"`
	Ned     string `xml:"ned"`
}

// HVPair holds an h/v attribute pair used in PVV fields.
type HVPair struct {
	H string `xml:"h,attr"`
	V string `xml:"v,attr"`
}

// Zal contains advance payment settings.
type Zal struct {
	Ved  string `xml:"ved,attr"`
	Pau  string `xml:"pau,attr"`
	VZ   string `xml:"vz,attr"`
	DP   string `xml:"dp,attr"`
	NP   string `xml:"np,attr"`
	Duch string `xml:"duch,attr"`
}

// Pre contains overpayment return information.
type Pre struct {
	Vra  string `xml:"vra,attr"`
	Kam  string `xml:"kam,attr"`
	Rok  string `xml:"rok"`
	IBAN string `xml:"iban"`
	BS   PreBS  `xml:"bs"`
	Adr  Address `xml:"adr"`
}

// PreBS contains bank account details for overpayment return.
type PreBS struct {
	PU string `xml:"pu,attr"`
	CU string `xml:"cu,attr"`
	KB string `xml:"kb,attr"`
	SS string `xml:"ss,attr"`
	VS string `xml:"vs,attr"`
}

// Prizn contains declaration flags.
type Prizn struct {
	Pau    string `xml:"pau"`
	Pov    string `xml:"pov"`
	Elektr string `xml:"elektr"`
	Por    string `xml:"por"`
	Meldat string `xml:"meldat"`
}

// Spo contains spouse information.
type Spo struct {
	Bno  string  `xml:"bno,attr"`
	Den  string  `xml:"den,attr"`
	Name SpoName `xml:"name"`
	Adr  Address `xml:"adr"`
}

// SpoName holds spouse name fields.
type SpoName struct {
	Sur string `xml:"sur,attr"`
	Fir string `xml:"fir,attr"`
	Tit string `xml:"tit,attr"`
}

// DatEl holds the filing date element.
type DatEl struct {
	Dre string `xml:"dre,attr"`
}
