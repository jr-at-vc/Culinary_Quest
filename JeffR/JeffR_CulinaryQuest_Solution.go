// the primary algorithm is in the findMostSatisfyingMeal function-
// see the godoc on and within that function for details
package main

//

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"slices"
	"strings"
)

// note the go.mod is used to support having the solution in a folder

// the name of the input menu json file;
//
//	define a variable here at the top for easy testing support-
//	NOTE this can also be overridden with -f={menuJsonFileSpec}
var menuInputFile string = "menu.json"

// ditto for user constraints; -u={userConstraintsFileSpec}
var userConstraintsInputFile string = "user_constraints.json"

// flag indicating whether to cleanse categories;
//
//	when enabled, the categories will be stripped of whitespace, converted to lowercase,
//	and stripped of pluralizing "s" suffix, if any,
//	allowing for minor-yet-insignificant variations in category names
var CLEANSE_CATEGORIES = true

// Culinary Quest readme sez all ingredients will be listed-
// we'll presume missing ingredients in old Best Meal samples means no allergens present
var CHECK_FOR_MISSING_INGREDIENTS = false

func main() {

	//
	// process command-line, if any;
	// if no parameters are used,
	// the default "menu.json" is expected to be in the current directory,
	// the only output will an error or best meal, both in json format.
	//

	// allow the menu.json input filespec to be customized without changing code
	menuFileParamPtr := flag.String("f", menuInputFile, "the input menu json filespec")
	// allow the user_constraints.json input filespec to be customized without changing code
	userConstraintsFileParamPtr := flag.String("u", userConstraintsInputFile, "the input menu json filespec")

	// cleanse parameter supports cleansing of category (ignore whitespace/case/pluralized)
	cleanseParamPtr := flag.Bool("c", CLEANSE_CATEGORIES, "whether to cleanse input categories")
	// verbose mode supports troubleshooting
	verboseParamPtr := flag.Bool("v", VERBOSE, "enable verbose output for troubleshooting")
	// the above *declares our support for command line params,
	// we must actually [flag.Parse]() to process all parameters
	flag.Parse()
	if verboseParamPtr != nil {
		// we had the -v parameter
		VERBOSE = *verboseParamPtr
		if VERBOSE {
			// -v or -v=true
			log.Printf("VERBOSE mode enabled.\n")
		}
	}
	if cleanseParamPtr != nil {
		CLEANSE_CATEGORIES = *cleanseParamPtr
	}

	if VERBOSE {
		log.Printf("Cleansing categories: %v\n", CLEANSE_CATEGORIES)
	}

	if menuFileParamPtr != nil {
		// we had the -f={menuJsonFileSpec} parameter
		menuInputFile = *menuFileParamPtr
		if VERBOSE {
			log.Printf("Reading menu json from `%s`\n", menuInputFile)
		}
	}

	if userConstraintsFileParamPtr != nil {
		// we had the -u={menuJsonFileSpec} parameter
		userConstraintsInputFile = *userConstraintsFileParamPtr
		if VERBOSE {
			log.Printf("Reading user constraints json from `%s`\n", userConstraintsInputFile)
		}
	}

	// find the best meal and report on it, else report an error
	findAndEmitBestMeal(menuInputFile, userConstraintsInputFile)
}

// the global-ish flag for verbose mode-
// if enabled, logs inner workings for algo
var VERBOSE = false

// find the best meal and report on it, else report an error
func findAndEmitBestMeal(menuInputFile string, userConstraintsInputFile string) {
	bestMeals, err := findBestMeal(menuInputFile, userConstraintsInputFile)

	if err != nil {
		emitBestMealError(err)
	} else {

		// the best meal will be the one-and-only item in the array returned
		bestMeal := bestMeals[0]

		emitBestMeal(bestMeal)
	}
}

// write the error that occurred while trying to find best meal to the console
func emitBestMealError(err error) {

	// construct the best meal error object
	bestMealError := BestMealError{err.Error()}

	//// original code to "manually" output in json:
	//// fmt.Printf("{\n\t\"error\": \"%s\"\n}\n", bestMealError.Error)

	// pretty print the output in json format, indenting with tabs
	json, _ := json.MarshalIndent(&bestMealError, "", "\t")
	fmt.Println(string(json))
}

// write the best meal we found to the console
func emitBestMeal(bestMeal BestMeal) {

	//// original code "manually" output best meal in json format
	/*
		fmt.Printf("{\n"+
			"\t\"selectedFoods\": %s,\n"+
			"\t\"totalCost\": %d,\n"+
			"\t\"totalSatisfaction\": %d\n"+
			"}\n",
			fmt.Sprintf("[ \"%s\", \"%s\", \"%s\", \"%s\" ]",
				bestMeal.SelectedFoods[APP_INDEX],
				bestMeal.SelectedFoods[DRINK_INDEX],
				bestMeal.SelectedFoods[MAIN_INDEX],
				bestMeal.SelectedFoods[DESSERT_INDEX],
			),
			bestMeal.TotalCost,
			bestMeal.TotalSatisfaction,
		)
	*/

	// pretty print the output in json format, indenting with tabs
	json, _ := json.MarshalIndent(&bestMeal, "", "\t")
	fmt.Println(string(json))

}

////////////////////////////////////////////////////////////////////
// define the output

type SelectedFood struct {
	Category string `json:"category"`

	Name string `json:"name"`
}

// the final result: the most satisfying meal within our budget aka the best meal
type BestMeal struct {
	// the names of the foods selected, one from each category
	SelectedFoods [MEAL_ITEMS]SelectedFood `json:"selectedFoods"`

	// the total cost of this meal in dollars
	TotalCost int `json:"totalCost"`

	// the total satisfaction score for this meal
	TotalSatisfaction int `json:"totalSatisfaction"`

	Calories int `json:"calories"`
}

/*
// a few error conditions
type BestMealErr int

const ( // BEST_MEAL_ERR
	// e.g. missing category
	BAD_INPUT BestMealErr = 1
	// nothing available within your budget
	TOO_POOR
)
*/

// the structure used to report errors- these can be input errors or no meal within our budget
type BestMealError struct {

	// a description of the specific error encountered
	Error string `json:"error"`
}

////////////////////////////////////////////////////////////////////
// define the input

// a MenuItem represents an entry in the menu.foods array
type MenuItem struct {

	// the name of this menu item e.g. "Steak"
	Name string `json:"name"`

	// the cost of this menu item in dollars
	Cost int `json:"cost"`

	// the satisfaction score of this menu item
	Satisfaction int `json:"satisfaction"`

	// the category of this menu item e.g. "Main Course"
	Category string `json:"category"`

	Ingredients []string `json:"ingredients"`

	Calories int `json:"calories"`
}

// define the structure of the menu input:
//
//	an array of foods + our budget
type Menu struct {

	// the set of foods, including their cost, satisfaction and category
	Foods []MenuItem `json:"foods"`

	// the budget we have to spend on a four-part meal from this menu in dollars
	// NOTE for Culinary quest, budget field moves to UserConstraints
	// /* Best Meal: */ Budget int `json:"budget"`
}

// define the structure for user constraints
type UserConstraints struct {
	Budget int `json:"budget"`

	Allergy []string `json:"allergy"`

	CalorieLimit int `json:"calorieLimit"`
}

// find the best meal based on menu and budget in the spec'd inputFile .json;
// we return an array for the BestMeal to allow for nil value when we have an error
func findBestMeal(menuInputFile string, userConstraintsInputFile string) ([]BestMeal, error) {

	menu, err := loadMenu(menuInputFile)

	if err != nil {
		return nil, err
	}

	// do some basic error checking on the input
	if len(menu.Foods) == 0 {
		return nil, errors.New("No food in menu??")
	}

	/*
		// Best Meal had budget in menu; Culinary Quest moves budget to User Constraints
		if menu.Budget <= 0 {
			return nil, errors.New("You need a budget")
		}
	*/

	userConstraints, err := loadUserConstraints(userConstraintsInputFile)

	if err != nil {
		return nil, err
	}

	if userConstraints.Budget <= 0 {
		return nil, errors.New("You need a budget")
	}

	if userConstraints.CalorieLimit <= 0 {
		return nil, errors.New("No calorie limit specified")
	}

	meals, err := findMostSatisfyingMeal(menu.Foods, userConstraints.Budget, userConstraints.CalorieLimit, userConstraints.Allergy)

	if err != nil {
		return nil, err
	}

	// the most satisfying meal will be the one-and-only item in the array returned
	mostSatisfyingMeal := meals[0]

	bestMeal := BestMeal{
		// Best meal output food names only
		// mealFoodNames(menu.Foods, mostSatisfyingMeal),
		// Culinary Quest includes food category + name in output
		selectedFoods(menu.Foods, mostSatisfyingMeal),
		mostSatisfyingMeal.totalCost, mostSatisfyingMeal.totalSatisfaction, mostSatisfyingMeal.totalCalories}

	return []BestMeal{bestMeal}, nil

}

// Best Meal: loadMenuAndBudget
func loadMenu(menuInputFile string) (Menu, error) {

	// (eventually) our menu: foods + budget
	menu := Menu{make([]MenuItem, 0)} /* Budget: */ // , 0 }

	//// originally had the menu hard-wired into the code for a q&d test
	/*
		menu.foods = append(menu.foods, MenuItem{"Fried Calamari", 6, 5, "Appetizer"})
		menu.foods = append(menu.foods, MenuItem{"Bruschetta", 4, 4, "Appetizer"})

		menu.foods = append(menu.foods, MenuItem{"Soda", 1, 1, "Drink"})
		menu.foods = append(menu.foods, MenuItem{"Beer", 3, 2, "Drink"})

		menu.foods = append(menu.foods, MenuItem{"Lasagna", 8, 7, "Main Course"})
		menu.foods = append(menu.foods, MenuItem{"Burger", 6, 5, "Main Course"})

		menu.foods = append(menu.foods, MenuItem{"Cheesecake", 4, 4, "Dessert"})
		menu.foods = append(menu.foods, MenuItem{"Ice Cream", 2, 2, "Dessert"})

		menu.budget = 25
	*/

	// given the limit on input (200 items),
	// should be no issue reading the entire text file into memory
	menuBytes, err := os.ReadFile(menuInputFile)

	if err != nil {
		// bad file or access issue
		return menu, err
	}

	// menuString := string(menuBytes)

	// convert the json to in-memory object representation
	err = json.Unmarshal(menuBytes, &menu)

	if err != nil {
		// some issue with the json
		err = errors.New("Bad menu json: " + err.Error())
	}

	// menu loaded from json, return it w/ no error
	return menu, err
}

func loadUserConstraints(userConstraintsInputFile string) (UserConstraints, error) {

	userConstraints := UserConstraints{}

	userConstraintsBytes, err := os.ReadFile(userConstraintsInputFile)

	if err != nil {
		// bad file or access issue
		return userConstraints, err
	}

	err = json.Unmarshal(userConstraintsBytes, &userConstraints)

	if err != nil {
		// some issue with the json
		err = errors.New("Bad user constraints json: " + err.Error())
	}

	return userConstraints, err

}

// the Meal struct is used to track a particular instance of a meal-
// the combination of a specific appetizer, drink, main course and dessert;
// for the food items, we're actually tracking the index in the original menu item
// rather than a separate copy.
//
// we also track the totalCost and totalSatisfaction,
// the sum of the corresponding fields from the four food items in this meal
type Meal struct {
	// menu.foods index for appetizer
	appIndex int
	// menu.foods index for drink
	drinkIndex int
	// menu.foods index for main course
	mainCourseIndex int
	// menu.foods index for dessert
	dessertIndex int
	// sum of menu.foods[meal.app|drink|mainCourse|dessert indices].cost
	totalCost int
	// sum of menu.foods[meal.app|drink|mainCourse|dessert indices].satisfaction
	totalSatisfaction int

	totalCalories int
}

// *the* solution algorithm- given a list of food items and budget in dollars,
// find the most satisfying aka best meal
//
//   - break down the foods by category
//     -- filter by allergens in ingredients
//     -- check for unknown categories
//   - validate each category has at least one food
//   - for each combination of foods, one from each category, aka a meal:
//     -- compute the total calories
//     -- if total calories fits out limit:
//     -- compute the total cost
//     -- if total cost fits our budget:
//     -- compute the total satisfaction
//     -- if total satisfaction is greater than prior most-satisfying max,
//     *OR* if total satisfaction equals and total cost is lower than prior most-satisfying meal,
//     -- save the meal as most-satisfying so far
//
// as noted in the solution-specific readme,
// I factored cost into the equation-
// given two meals with the same total satisfaction,
// the meal with the lower total cost will be favored.
//
// we return an array for the meal so we can use nil when an error occurs
func findMostSatisfyingMeal(foods []MenuItem, budget int, calorieLimit int, allergies []string) ([]Meal, error) {

	if /* foods == nil || */ len(foods) == 0 {
		return nil, errors.New("Nothing in the menu??")
	}

	// "The number of items per category on the menu can range from 0 to 50." -- main README from Alek
	// with a max of 50 items in each of 4 categories, the max possible meals would be 50^4 or 6.25M

	// assemble the list of food items by category;
	// the values we're tracking here are indexes in the set of foods
	appIndexes := make([]int, 0)
	drinkIndexes := make([]int, 0)
	mainCourseIndexes := make([]int, 0)
	dessertIndexes := make([]int, 0)

	// detect duplicate names- Best Meal only
	// /* NO>>> */ knownFoodNames := make(map[string]int, 0)

	rejectedCount := 0

	for i, food := range foods {

		if VERBOSE {
			log.Printf("Input foods[%d]: %+v\n", i, food)
		}

		// per the Culinary Quest readme, we're not concerned re duplicates-
		// a) output includes food category, so no concern re duplicate names across categories
		// b) "Each dish name within a category will be unique."
		/*
			if knownFoodNames[food.Name] > 0 {
				return nil, fmt.Errorf("Duplicate food name=`%s` at foods[%d] and foods[%d]", food.Name, knownFoodNames[food.Name]-1, i)
			}

			knownFoodNames[food.Name] = i + 1
		*/

		// check for allergens
		if len(allergies) > 0 {
			rejected := false
			if len(food.Ingredients) > 0 {
				slices.Sort(food.Ingredients)

				for _, allergy := range allergies {
					matchIndex, matchFound := slices.BinarySearchFunc(food.Ingredients, allergy, func(ingredient string, allergy string) int {
						return strings.Compare(strings.ToLower(ingredient), strings.ToLower(allergy))
					})

					if matchFound {
						if VERBOSE {
							log.Printf("Food `%s` contains allergen `%s`- rejected\n", food.Name, food.Ingredients[matchIndex])
						}
						rejected = true
						break
					}
				}

			} else {

				if CHECK_FOR_MISSING_INGREDIENTS {
					if VERBOSE {
						log.Printf("User has allergies but food `%s` lists no ingredients- rejected for safety\n", food.Name)
					}

					rejected = true

				} else {
					if VERBOSE {
						log.Printf("User has allergies but food `%s` lists no ingredients- presuming no allergens\n", food.Name)
					}
				}
			}

			if rejected {
				rejectedCount++
				continue
			}

		}

		switch cleanseCategory(food.Category) {
		case cleanseCategory(APPETIZER_CATEGORY_NAME):
			{
				appIndexes = append(appIndexes, i)
			}
		case cleanseCategory(DRINK_CATEGORY_NAME):
			{
				drinkIndexes = append(drinkIndexes, i)
			}
		case cleanseCategory(MAIN_COURSE_CATEGORY_NAME):
			{
				mainCourseIndexes = append(mainCourseIndexes, i)
			}
		case cleanseCategory(DESSERT_CATEGORY_NAME):
			{
				dessertIndexes = append(dessertIndexes, i)
			}
		default:
			return nil, fmt.Errorf("Unknown foods[%d] category: %+v", i, food)
		}
	}

	// verify we have something in each category- note only the first gap found is reported
	if len(appIndexes) == 0 {
		return nil, errors.New("No (compatible?) apps in menu")
	}

	if len(drinkIndexes) == 0 {
		return nil, errors.New("No (compatible?) drinks in menu")
	}

	if len(mainCourseIndexes) == 0 {
		return nil, errors.New("No (compatible?) mains in menu")
	}

	if len(dessertIndexes) == 0 {
		return nil, errors.New("No (compatible?) desserts in menu")
	}

	if VERBOSE {
		log.Printf("Checking %d apps x %d drinks x %d mains x %d desserts = %d meals for $%d budget\n",
			len(appIndexes), len(drinkIndexes), len(mainCourseIndexes), len(dessertIndexes),
			len(appIndexes)*len(drinkIndexes)*len(mainCourseIndexes)*len(dessertIndexes),
			budget)
	}

	// now that we have validated input broken down by category,
	// find the most satisfying meal within our budget

	// have we found at least one candidate
	foundMostSatisfyingMeal := false
	// track the maximum total satisfaction we've found so far
	maxTotalSatisfaction := 0
	// track the total cost of most satisfying meal so far;
	// we factor this into the satisfaction, lower cost is more satisfying
	lowestSatisfyingTotalCost := math.MaxInt

	// track the total calories of the most satisfying meal so far;
	// we factor this into the satisfaction, lower calories is more satisfying
	// /* NO>>> lower calories != greater satisfaction */ lowestSatisfyingTotalCalories := math.MaxInt

	// track the cheapest four-part meal regardless of budget;
	// if no meal is available within budget,
	// we'll let them know how many $s short they are
	cheapestMealTotalCost := math.MaxInt

	// track the lowest calorie meal (while respecting allergens);
	// if no meal is available within calorie limit,
	// we'll let them know how far off they are
	lowestCalorieMeal := math.MaxInt

	// track the actual most satisfying meal within budget found so far
	// as we work our way thru the menu food items
	var mostSatisfyingMeal Meal

	//
	mealCounter := 0

	// for each combination of foods, one from each category

	for _, appIndex := range appIndexes {
		for _, drinkIndex := range drinkIndexes {
			for _, mainCourseIndex := range mainCourseIndexes {
				for _, dessertIndex := range dessertIndexes {

					if VERBOSE {
						log.Printf("Meal #%d foods[] indexes: app=[%d] drink=[%d] main=[%d] dessert=[%d]\n",
							mealCounter, appIndex, drinkIndex, mainCourseIndex, dessertIndex)
					}

					totalCalories := 0 +
						foods[appIndex].Calories +
						foods[drinkIndex].Calories +
						foods[mainCourseIndex].Calories +
						foods[dessertIndex].Calories

					if totalCalories <= calorieLimit {

						totalCost := 0 +
							foods[appIndex].Cost +
							foods[drinkIndex].Cost +
							foods[mainCourseIndex].Cost +
							foods[dessertIndex].Cost

						if totalCost <= budget {

							// this meal is within budget
							// compute our total satisfaction for this meal

							totalSatisfaction := 0 +
								foods[appIndex].Satisfaction +
								foods[drinkIndex].Satisfaction +
								foods[mainCourseIndex].Satisfaction +
								foods[dessertIndex].Satisfaction

							// highest satisfaction
							if totalSatisfaction > maxTotalSatisfaction ||
								// equal satisfaction but lower cost (which is also satisfying)
								(totalSatisfaction >= maxTotalSatisfaction && totalCost < lowestSatisfyingTotalCost) { // ||
								// NOTE per the Culinary Quest readme,
								//  "If there is a tie in satisfaction, the cheaper combination should be chosen."
								// i.e. lower calories does not result in greater satisfaction
								// (I beg to differ, but thems the rules)
								// /* NO>>> */ equal satisfaction but lower calories (which is also satisfying)
								// /* NO>>> */ (totalSatisfaction >= maxTotalSatisfaction && totalCalories < lowestSatisfyingTotalCalories) {

								// save off the candidate meal and stats
								mostSatisfyingMeal = Meal{appIndex, drinkIndex, mainCourseIndex, dessertIndex, totalCost, totalSatisfaction, totalCalories}
								maxTotalSatisfaction = totalSatisfaction
								lowestSatisfyingTotalCost = totalCost
								// /* NO>>> lower calories != greater satisfaction */ lowestSatisfyingTotalCalories = totalCalories
								foundMostSatisfyingMeal = true

								// since we found at least one candidate, we'll no longer be tracking minimum cost or lowest calories
								cheapestMealTotalCost = -1
								lowestCalorieMeal = -1

								if VERBOSE {
									log.Printf("** Most Satisfying + Lowest Cost & Calories (so far): %s totalCost=%d totalSatisfaction=%d totalCalories=%d\n", foodNames(foods, appIndex, drinkIndex, mainCourseIndex, dessertIndex), totalCost, totalSatisfaction, totalCalories)
								}
							} else {
								if VERBOSE {
									log.Printf("Less Satisfying: %s totalCost=%d totalSatisfaction=%d totalCalories=%d\n", foodNames(foods, appIndex, drinkIndex, mainCourseIndex, dessertIndex), totalCost, totalSatisfaction, totalCalories)
								}
							}

						} else {
							if VERBOSE {
								log.Printf("Over budget: %s totalCost=%d\n", foodNames(foods, appIndex, drinkIndex, mainCourseIndex, dessertIndex), totalCost)
							}
						}

						if !foundMostSatisfyingMeal {
							if totalCost < cheapestMealTotalCost {
								cheapestMealTotalCost = totalCost
								if VERBOSE {
									log.Printf("Cheapest meal cost: %d\n", cheapestMealTotalCost)
								}
							}
						}

					} else {
						if VERBOSE {
							log.Printf("Over calories: %s totalCalories=%d\n", foodNames(foods, appIndex, drinkIndex, mainCourseIndex, dessertIndex), totalCalories)
						}
					}

					if !foundMostSatisfyingMeal {
						if totalCalories < lowestCalorieMeal {
							lowestCalorieMeal = totalCalories
							if VERBOSE {
								log.Printf("Lowest calorie meal: %d\n", lowestCalorieMeal)
							}
						}
					}

					mealCounter++
				}
			}
		}
	}

	if foundMostSatisfyingMeal {
		if VERBOSE {
			log.Printf("**** Final Most Satisfying + Lowest Cost & Calories Meal: %s totalCost=%d totalSatisfaction=%d totalCalories=%d\n",
				foodNames(foods,
					mostSatisfyingMeal.appIndex,
					mostSatisfyingMeal.drinkIndex,
					mostSatisfyingMeal.mainCourseIndex,
					mostSatisfyingMeal.dessertIndex),
				mostSatisfyingMeal.totalCost,
				mostSatisfyingMeal.totalSatisfaction,
				mostSatisfyingMeal.totalCalories)
		}
		return []Meal{mostSatisfyingMeal}, nil
	} else {
		if cheapestMealTotalCost != math.MaxInt {

			if VERBOSE {
				log.Printf("*** Budget=%d vs Cheapest Meal=%d\n", budget, cheapestMealTotalCost)
			}

			return nil,
				fmt.Errorf(""+
					"Checked %d meal(s) excluding %d allergen-containg foods, none fit your budget- "+
					"you need another %d buck(s) to dine here, you poor bleepard :/",
					mealCounter, rejectedCount, cheapestMealTotalCost-budget)

		} else {

			if VERBOSE {
				log.Printf("*** CalorieLimit=%d vs Lowest Calorie Meal=%d\n", calorieLimit, lowestCalorieMeal)
			}

			return nil,
				fmt.Errorf(""+
					"Checked %d meal(s) excluding %d allergen-containg foods, none fit your calorie limit- "+
					"you need another %d calorie(s) to dine here, you fat bleepard :/",
					mealCounter, rejectedCount, lowestCalorieMeal-calorieLimit)

		}
	}
}

func cleanseCategory(category string) string {
	if CLEANSE_CATEGORIES {
		// ignore whitespace, ignore case, ignore plural vs singular
		cleansed := strings.TrimSuffix(strings.ToLower(strings.ReplaceAll(category, " ", "")), "s")
		return cleansed
	} else {
		return category // uncleansed
	}
}

func mealFoodNames(foods []MenuItem, meal Meal) [MEAL_ITEMS]string {
	return foodNames(foods,
		meal.appIndex,
		meal.drinkIndex,
		meal.mainCourseIndex,
		meal.dessertIndex)
}

func selectedFoods(foods []MenuItem, meal Meal) [MEAL_ITEMS]SelectedFood {
	return [MEAL_ITEMS]SelectedFood{
		{foods[meal.appIndex].Category, foods[meal.appIndex].Name},
		{foods[meal.drinkIndex].Category, foods[meal.drinkIndex].Name},
		{foods[meal.mainCourseIndex].Category, foods[meal.mainCourseIndex].Name},
		{foods[meal.dessertIndex].Category, foods[meal.dessertIndex].Name},
	}
}

// convert the indexes for foods in a four-part meal to the food names
func foodNames(foods []MenuItem, appIndex int, drinkIndex int, mainCourseIndex int, dessertIndex int) [MEAL_ITEMS]string {
	return [MEAL_ITEMS]string{
		foods[appIndex].Name,
		foods[drinkIndex].Name,
		foods[mainCourseIndex].Name,
		foods[dessertIndex].Name,
	}
}

// the food type names used to categorize the foods in the menu
const ( // MenuItemCategory
	APPETIZER_CATEGORY_NAME   string = "Appetizer"
	DRINK_CATEGORY_NAME       string = "Drink"
	MAIN_COURSE_CATEGORY_NAME string = "Main Course"
	DESSERT_CATEGORY_NAME     string = "Dessert"
)

const ( // MenuItemCategoryIndex
	APP_INDEX = iota
	DRINK_INDEX
	MAIN_INDEX
	DESSERT_INDEX
	MEAL_ITEMS
)
