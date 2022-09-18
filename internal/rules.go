package internal

import (
	"regexp"

	"github.com/deckarep/golang-set"
)

type nameRule struct {
	Name        string
	DisplayName string
	ColumnNames []string
}

type multiNameRule struct {
	Name        string
	DisplayName string
	ColumnNames [][]string
}

type regexRule struct {
	Name        string
	DisplayName string
	Regex       *regexp.Regexp
}

type tokenRule struct {
	Name        string
	DisplayName string
	Tokens      mapset.Set
}

// columns are lowercased and _ are removed
// this allows use a single list for under_score and camelCase
// no rules for email or IP, since they can be detected automatically
// keep last name and phone until better international support
var nameRules = []nameRule{
	nameRule{Name: "surname", DisplayName: "last names", ColumnNames: []string{"lastname", "lname", "surname"}},
	nameRule{Name: "phone", DisplayName: "phone numbers", ColumnNames: []string{"phone", "phonenumber"}},
	nameRule{Name: "date_of_birth", DisplayName: "dates of birth", ColumnNames: []string{"dateofbirth", "birthday", "dob"}},
	nameRule{Name: "postal_code", DisplayName: "postal codes", ColumnNames: []string{"zip", "zipcode", "postalcode"}},
	nameRule{Name: "oauth_token", DisplayName: "OAuth tokens", ColumnNames: []string{"accesstoken", "refreshtoken"}},
}

var multiNameRules = []multiNameRule{
	multiNameRule{Name: "location", DisplayName: "location data", ColumnNames: [][]string{{"latitude", "lat"}, {"longitude", "lon", "lng"}}},
}

// TODO IPv6
// TODO more popular access tokens
var regexRules = []regexRule{
	regexRule{Name: "email", DisplayName: "emails", Regex: regexp.MustCompile(`\b[\w][\w+.-]+(@|%40)[a-z\d-]+(\.[a-z\d-]+)*\.[a-z]+\b`)},
	regexRule{Name: "ip", DisplayName: "IP addresses", Regex: regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)},
	regexRule{Name: "credit_card", DisplayName: "credit card numbers", Regex: regexp.MustCompile(`(\b[3456]\d{3}[\s+-]\d{4}[\s+-]\d{4}[\s+-]\d{4}\b)|(\b[3456]\d{15}\b)`)},
	regexRule{Name: "phone", DisplayName: "phone numbers", Regex: regexp.MustCompile(`(\b(\+\d{1,2}\s)?\(?\d{3}\)?[\s+.-]\d{3}[\s+.-]\d{4}\b)|((?:\+|%2B)[1-9]\d{6,14}\b)`)},
	regexRule{Name: "ssn", DisplayName: "SSNs", Regex: regexp.MustCompile(`\b\d{3}[\s+-]\d{2}[\s+-]\d{4}\b`)},
	regexRule{Name: "street", DisplayName: "street addresses", Regex: regexp.MustCompile(`(?i)\b\d+\b.{4,60}\b(st|street|ave|avenue|road|rd|drive|dr)\b`)},
	regexRule{Name: "oauth_token", DisplayName: "OAuth tokens", Regex: regexp.MustCompile(`ya29\..{60,200}`)}, // google
	regexRule{Name: "mac", DisplayName: "MAC addresses", Regex: regexp.MustCompile(`\b[0-9a-fA-F]{2}(?:(?::|%3A)[0-9a-fA-F]{2}){5}\b`)},
}

// first 300 from 2010 US Census https://www.census.gov/topics/population/genealogy/data/2010_surnames.html
// first 300 covered ~30% cumulative density inn 1990 US Census
var lastNames = []interface{}{"smith", "johnson", "williams", "brown", "jones", "garcia", "miller", "davis", "rodriguez", "martinez", "hernandez", "lopez", "gonzalez", "wilson", "anderson", "thomas", "taylor", "moore", "jackson", "martin", "lee", "perez", "thompson", "white", "harris", "sanchez", "clark", "ramirez", "lewis", "robinson", "walker", "young", "allen", "king", "wright", "scott", "torres", "nguyen", "hill", "flores", "green", "adams", "nelson", "baker", "hall", "rivera", "campbell", "mitchell", "carter", "roberts", "gomez", "phillips", "evans", "turner", "diaz", "parker", "cruz", "edwards", "collins", "reyes", "stewart", "morris", "morales", "murphy", "cook", "rogers", "gutierrez", "ortiz", "morgan", "cooper", "peterson", "bailey", "reed", "kelly", "howard", "ramos", "kim", "cox", "ward", "richardson", "watson", "brooks", "chavez", "wood", "james", "bennett", "gray", "mendoza", "ruiz", "hughes", "price", "alvarez", "castillo", "sanders", "patel", "myers", "long", "ross", "foster", "jimenez", "powell", "jenkins", "perry", "russell", "sullivan", "bell", "coleman", "butler", "henderson", "barnes", "gonzales", "fisher", "vasquez", "simmons", "romero", "jordan", "patterson", "alexander", "hamilton", "graham", "reynolds", "griffin", "wallace", "moreno", "west", "cole", "hayes", "bryant", "herrera", "gibson", "ellis", "tran", "medina", "aguilar", "stevens", "murray", "ford", "castro", "marshall", "owens", "harrison", "fernandez", "mcdonald", "woods", "washington", "kennedy", "wells", "vargas", "henry", "chen", "freeman", "webb", "tucker", "guzman", "burns", "crawford", "olson", "simpson", "porter", "hunter", "gordon", "mendez", "silva", "shaw", "snyder", "mason", "dixon", "munoz", "hunt", "hicks", "holmes", "palmer", "wagner", "black", "robertson", "boyd", "rose", "stone", "salazar", "fox", "warren", "mills", "meyer", "rice", "schmidt", "garza", "daniels", "ferguson", "nichols", "stephens", "soto", "weaver", "ryan", "gardner", "payne", "grant", "dunn", "kelley", "spencer", "hawkins", "arnold", "pierce", "vazquez", "hansen", "peters", "santos", "hart", "bradley", "knight", "elliott", "cunningham", "duncan", "armstrong", "hudson", "carroll", "lane", "riley", "andrews", "alvarado", "ray", "delgado", "berry", "perkins", "hoffman", "johnston", "matthews", "pena", "richards", "contreras", "willis", "carpenter", "lawrence", "sandoval", "guerrero", "george", "chapman", "rios", "estrada", "ortega", "watkins", "greene", "nunez", "wheeler", "valdez", "harper", "burke", "larson", "santiago", "maldonado", "morrison", "franklin", "carlson", "austin", "dominguez", "carr", "lawson", "jacobs", "obrien", "lynch", "singh", "vega", "bishop", "montgomery", "oliver", "jensen", "harvey", "williamson", "gilbert", "dean", "sims", "espinoza", "howell", "li", "wong", "reid", "hanson", "le", "mccoy", "garrett", "burton", "fuller", "wang", "weber", "welch", "rojas", "lucas", "marquez", "fields", "park", "yang", "little", "banks", "padilla", "day", "walsh", "bowman", "schultz", "luna", "fowler", "mejia"}
var tokenRules = []tokenRule{
	tokenRule{Name: "surname", DisplayName: "last names", Tokens: mapset.NewSetFromSlice(lastNames)},
}
