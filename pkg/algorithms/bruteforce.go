package algorithms

/*
*	Bruteforce method for calculating Knapsack problem
 */
func KnapsackBruteForce(capacity int, items []FItem, indexes []int, lastIndex int, sumWeight int, sumValue float64) (int, float64, []int) {

	var bestWeight = sumWeight
	var bestValue = sumValue
	var bestConfiguration []int = indexes

	if lastIndex == len(items) { return sumWeight, sumValue, indexes }

	for i := lastIndex; i < len(items); i++ {
		weight, value, configuration := KnapsackBruteForce(capacity, items, append(indexes, i), i + 1, sumWeight + items[i].Weight, sumValue + items[i].Value)
		if value > bestValue && weight <= capacity{
			// fmt.Println("weight",weight, "cap", capacity, "config", configuration)
			bestValue = value
			bestConfiguration = configuration
		}
	}
	
	return bestWeight ,bestValue, bestConfiguration
}

