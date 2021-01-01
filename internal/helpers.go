package internal

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/deckarep/golang-set"
	"github.com/fatih/color"
)

type nameRule struct {
	Name        string
	DisplayName string
	ColumnNames []string
}

type regexRule struct {
	Name        string
	DisplayName string
	Regex       *regexp.Regexp
}

type ruleMatch struct {
	RuleName    string
	DisplayName string
	Confidence  string
	Identifier  string
	MatchedData []string
	MatchType   string
}

func unique(arr []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range arr {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func anyMatches(a []string) bool {
	for _, av := range a {
		if lastNamesSet.Contains(av) {
			return true
		}
	}
	return false
}

func matchNameRule(name string, rules []nameRule) nameRule {
	for _, rule := range rules {
		if stringInSlice(name, rule.ColumnNames) {
			return rule
		}
	}
	return nameRule{}
}

// columns are lowercased and _ are removed
// this allows use a single list for under_score and camelCase
// no rules for email or IP, since they can be detected automatically
// keep last name and phone until better international support
var nameRules = []nameRule{
	nameRule{Name: "last_name", DisplayName: "last names", ColumnNames: []string{"lastname", "lname", "surname"}},
	nameRule{Name: "phone", DisplayName: "phone numbers", ColumnNames: []string{"phone", "phonenumber"}},
	nameRule{Name: "date_of_birth", DisplayName: "dates of birth", ColumnNames: []string{"dateofbirth", "birthday", "dob"}},
	nameRule{Name: "postal_code", DisplayName: "postal codes", ColumnNames: []string{"zip", "zipcode", "postalcode"}},
	nameRule{Name: "oauth_token", DisplayName: "OAuth tokens", ColumnNames: []string{"accesstoken", "refreshtoken"}},
}

// TODO IPv6
// TODO more popular access tokens
var regexRules = []regexRule{
	regexRule{
        Name: "email",
        DisplayName: "emails",
        Regex: regexp.MustCompile(`\b[\w][\w+.-]+(@|%40)[a-z\d-]+(\.[a-z\d-]+)*\.[a-z]+\b`),
    },
	regexRule{
        Name: "ip",
        DisplayName: "IP addresses",
        Regex: regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
    },
    regexRule{
        Name: "mac",
        DisplayName: "MAC addresses",
        Regex: regexp.MustCompile(`\b([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})\b`),
    },
	regexRule{
        Name: "credit_card",
        DisplayName: "credit card numbers",
        Regex: regexp.MustCompile(`\b[3456]\d{3}[\s+-]\d{4}[\s+-]\d{4}[\s+-]\d{4}\b`),
    },
	regexRule{
        Name: "credit_card",
        DisplayName: "credit card numbers",
        Regex: regexp.MustCompile(`\b[3456]\d{15}\b`),
    },
	regexRule{
        Name: "phone",
        DisplayName: "phone numbers",
        Regex: regexp.MustCompile(`\b(\+\d{1,2}\s)?\(?\d{3}\)?[\s+.-]\d{3}[\s+.-]\d{4}\b`),
    },
	regexRule{
        Name: "ssn",
        DisplayName: "SSNs",
        Regex: regexp.MustCompile(`\b\d{3}[\s+-]\d{2}[\s+-]\d{4}\b`),
    },
	regexRule{
        Name: "street",
        DisplayName: "street addresses",
        Regex: regexp.MustCompile(`(?i)\b\d+\b.{4,60}\b(st|street|ave|avenue|road|rd|drive|dr)\b`),
    },
	regexRule{
        Name: "oauth_token",
        DisplayName: "OAuth tokens",
        Regex: regexp.MustCompile(`ya29\..{60,200}`),
    }, // google
}

// first 300 from 2010 US Census https://www.census.gov/topics/population/genealogy/data/2010_surnames.html
// first 300 covered ~30% cumulative density inn 1990 US Census
var lastNames = []interface{}{"smith", "johnson", "williams", "brown", "jones", "garcia", "miller", "davis", "rodriguez", "martinez", "hernandez", "lopez", "gonzalez", "wilson", "anderson", "thomas", "taylor", "moore", "jackson", "martin", "lee", "perez", "thompson", "white", "harris", "sanchez", "clark", "ramirez", "lewis", "robinson", "walker", "young", "allen", "king", "wright", "scott", "torres", "nguyen", "hill", "flores", "green", "adams", "nelson", "baker", "hall", "rivera", "campbell", "mitchell", "carter", "roberts", "gomez", "phillips", "evans", "turner", "diaz", "parker", "cruz", "edwards", "collins", "reyes", "stewart", "morris", "morales", "murphy", "cook", "rogers", "gutierrez", "ortiz", "morgan", "cooper", "peterson", "bailey", "reed", "kelly", "howard", "ramos", "kim", "cox", "ward", "richardson", "watson", "brooks", "chavez", "wood", "james", "bennett", "gray", "mendoza", "ruiz", "hughes", "price", "alvarez", "castillo", "sanders", "patel", "myers", "long", "ross", "foster", "jimenez", "powell", "jenkins", "perry", "russell", "sullivan", "bell", "coleman", "butler", "henderson", "barnes", "gonzales", "fisher", "vasquez", "simmons", "romero", "jordan", "patterson", "alexander", "hamilton", "graham", "reynolds", "griffin", "wallace", "moreno", "west", "cole", "hayes", "bryant", "herrera", "gibson", "ellis", "tran", "medina", "aguilar", "stevens", "murray", "ford", "castro", "marshall", "owens", "harrison", "fernandez", "mcdonald", "woods", "washington", "kennedy", "wells", "vargas", "henry", "chen", "freeman", "webb", "tucker", "guzman", "burns", "crawford", "olson", "simpson", "porter", "hunter", "gordon", "mendez", "silva", "shaw", "snyder", "mason", "dixon", "munoz", "hunt", "hicks", "holmes", "palmer", "wagner", "black", "robertson", "boyd", "rose", "stone", "salazar", "fox", "warren", "mills", "meyer", "rice", "schmidt", "garza", "daniels", "ferguson", "nichols", "stephens", "soto", "weaver", "ryan", "gardner", "payne", "grant", "dunn", "kelley", "spencer", "hawkins", "arnold", "pierce", "vazquez", "hansen", "peters", "santos", "hart", "bradley", "knight", "elliott", "cunningham", "duncan", "armstrong", "hudson", "carroll", "lane", "riley", "andrews", "alvarado", "ray", "delgado", "berry", "perkins", "hoffman", "johnston", "matthews", "pena", "richards", "contreras", "willis", "carpenter", "lawrence", "sandoval", "guerrero", "george", "chapman", "rios", "estrada", "ortega", "watkins", "greene", "nunez", "wheeler", "valdez", "harper", "burke", "larson", "santiago", "maldonado", "morrison", "franklin", "carlson", "austin", "dominguez", "carr", "lawson", "jacobs", "obrien", "lynch", "singh", "vega", "bishop", "montgomery", "oliver", "jensen", "harvey", "williamson", "gilbert", "dean", "sims", "espinoza", "howell", "li", "wong", "reid", "hanson", "le", "mccoy", "garrett", "burton", "fuller", "wang", "weber", "welch", "rojas", "lucas", "marquez", "fields", "park", "yang", "little", "banks", "padilla", "day", "walsh", "bowman", "schultz", "luna", "fowler", "mejia"}
var lastNamesSet = mapset.NewSetFromSlice(lastNames)
var tokenizer = regexp.MustCompile(`\W+`)

var space = regexp.MustCompile(`\s+`)
var urlPassword = regexp.MustCompile(`((\/\/|%2F%2F)\S+(:|%3A))\S+(@|%40)`)

func findMatches(values []string) ([][]string, int) {
	matchedValues := make([][]string, len(regexRules)+1)
	nameIndex := len(regexRules)

	for _, v := range values {
		for i, rule := range regexRules {
			if rule.Regex.MatchString(v) {
				matchedValues[i] = append(matchedValues[i], v)
			}
		}

		tokens := tokenizer.Split(strings.ToLower(v), -1)
		if anyMatches(tokens) {
			matchedValues[nameIndex] = append(matchedValues[nameIndex], v)
		}
	}

	return matchedValues, len(values)
}

func checkMatches(colIdentifier string, matchedValues [][]string, count int, onlyValues bool) []ruleMatch {
	matchList := []ruleMatch{}

	for i, rule := range regexRules {
		matchedData := matchedValues[i]

		if rule.Name == "email" {
			// filter out false positives with URL credentials
			newMatchedData := matchedData
			matchedData = []string{}
			for _, v := range newMatchedData {
				// replace urls and check for email match again
				v2 := urlPassword.ReplaceAllString(v, "[FILTERED]")
				if rule.Regex.MatchString(v2) {
					matchedData = append(matchedData, v)
				}
			}
		}

		if len(matchedData) > 0 {
			confidence := "low"
			if rule.Name == "email" || float64(len(matchedData))/float64(count) > 0.5 {
				confidence = "high"
			}

			if onlyValues {
				var matchedValues []string
				for _, v := range matchedData {
					v3 := rule.Regex.FindAllString(v, -1)
					matchedValues = append(matchedValues, v3...)
				}
				matchedData = matchedValues
			}

			matchList = append(matchList, ruleMatch{RuleName: rule.Name, DisplayName: rule.DisplayName, Confidence: confidence, Identifier: colIdentifier, MatchedData: matchedData})
		}
	}

	// find names
	nameIndex := len(regexRules)
	matchedData := matchedValues[nameIndex]

	if len(matchedData) > 0 {
		confidence := "low"
		if float64(len(matchedData))/float64(count) > 0.1 && len(unique(matchedData)) >= 10 {
			confidence = "high"
		}

		if onlyValues {
			var matchedValues []string
			for _, v := range matchedData {
				tokens := tokenizer.Split(strings.ToLower(v), -1)
				for _, v2 := range tokens {
					if lastNamesSet.Contains(v2) {
						matchedValues = append(matchedValues, v2)
					}
				}
			}
			matchedData = matchedValues
		}

		matchList = append(matchList, ruleMatch{RuleName: "last_name", DisplayName: "last names", Confidence: confidence, Identifier: colIdentifier, MatchedData: matchedData})
	}

	return matchList
}

func checkTableData(table table, columnNames []string, columnValues [][]string) []ruleMatch {
	tableMatchList := []ruleMatch{}

	for i, col := range columnNames {
		// check values
		values := columnValues[i]
		colIdentifier := table.displayName() + "." + col

		matchedValues, count := findMatches(values)
		matchList := checkMatches(colIdentifier, matchedValues, count, false)

		// only check name if no matches
		if len(matchList) == 0 {
			name := strings.Replace(strings.ToLower(col), "_", "", -1)

			rule := matchNameRule(name, nameRules)
			if rule.Name != "" {
				matchList = append(matchList, ruleMatch{RuleName: rule.Name, DisplayName: rule.DisplayName, Confidence: "medium", Identifier: colIdentifier, MatchedData: values, MatchType: "name"})
			}
		}

		tableMatchList = append(tableMatchList, matchList...)
	}

	// check for location data
	var latCol string
	var lonCol string
	for _, col := range columnNames {
		if stringInSlice(col, []string{"latitude", "lat"}) {
			latCol = col
		} else if stringInSlice(col, []string{"longitude", "lon", "lng"}) {
			lonCol = col
		}
	}
	if latCol != "" && lonCol != "" {
		// TODO show data
		tableMatchList = append(tableMatchList, ruleMatch{RuleName: "location", DisplayName: "location data", Confidence: "medium", Identifier: table.displayName() + "." + latCol + "+" + lonCol, MatchType: "name"})
	}

	return tableMatchList
}

func pluralize(count int, singular string) string {
	if count != 1 {
		if strings.HasSuffix(singular, "ch") {
			singular = singular + "es"
		} else {
			singular = singular + "s"
		}
	}
	return fmt.Sprintf("%d %s", count, singular)
}

func printMatchList(matchList []ruleMatch, showData bool, showAll bool, rowStr string) {
	// print matches for table
	for _, match := range matchList {
		if showAll || match.Confidence != "low" {
			var description string

			count := len(match.MatchedData)
			if match.MatchType == "name" {
				description = fmt.Sprintf("possible %s (name match)", match.DisplayName)
			} else {
				str := pluralize(count, rowStr)
				if match.Confidence == "low" {
					str = str + ", low confidence"
				}
				description = fmt.Sprintf("found %s (%s)", match.DisplayName, str)
			}

			yellow := color.New(color.FgYellow).SprintFunc()
			fmt.Printf("%s %s\n", yellow(match.Identifier+":"), description)

			if showData {
				v := unique(match.MatchedData)
				if len(v) > 0 && showData {
					if len(v) > 50 {
						v = v[0:50]
					}

					for i, v2 := range v {
						v[i] = space.ReplaceAllString(v2, " ")
					}
					sort.Strings(v)
					fmt.Println("    " + strings.Join(v, ", "))
				}
				fmt.Println("")
			}
		}
	}
}

func showLowConfidenceMatchHelp(matchList []ruleMatch) {
	lowConfidenceMatches := []ruleMatch{}
	for _, match := range matchList {
		if match.Confidence == "low" {
			lowConfidenceMatches = append(lowConfidenceMatches, match)
		}
	}
	if len(lowConfidenceMatches) > 0 {
		fmt.Println("Also found " + pluralize(len(lowConfidenceMatches), "low confidence match") + ". Use --show-all to view them")
	}
}

func abort(err error) {
	fmt.Println(err)
	os.Exit(1)
}
