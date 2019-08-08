package main

import "testing"

func TestCircularQueueProgression(t *testing.T) {
	q := NewSimpleStringCircularBuffer(5)

	if !q.HasNoItems() {
		t.Errorf("On empty queue, HasNoItems returns false")
	}

	if q.NumberOfItemsInTheQueue() != 0 {
		t.Errorf("On empty queue, expect NumberOfItemsInTheQueue = 0, got = %d", q.NumberOfItemsInTheQueue())
	}

	stringSet := []string{"one", "two", "three", "four", "five"}
	for expectedInsertIndex, value := range stringSet {
		q.PutItemAtEnd(value)

		if q.HasNoItems() {
			t.Errorf("On insert number %d, HasNoItems() returns true", expectedInsertIndex+1)
		}

		if q.NumberOfItemsInTheQueue() != uint(expectedInsertIndex)+1 {
			t.Errorf("On insert number %d, expected NumberOfItemsInTheQueue() == %d, got = %d", expectedInsertIndex+1, expectedInsertIndex+1, q.NumberOfItemsInTheQueue())
		}

		for i := uint(0); i <= uint(expectedInsertIndex); i++ {
			item, hasItemAtThisIndex := q.GetItemAtIndex(i)

			if !hasItemAtThisIndex {
				t.Errorf("After insert at position %d expected item at position %d but GetItemAtIndex() returns false", expectedInsertIndex, i)
			}

			if item != stringSet[i] {
				t.Errorf("After insert at position %d expected item at position %d to be (%s), got (%s)", expectedInsertIndex, i, stringSet[i], item)
			}
		}

		item, hasItemAtThisIndex := q.GetItemAtIndex(uint(expectedInsertIndex) + 1)

		if hasItemAtThisIndex {
			t.Errorf("After insert at position %d expected no item at position %d but GetItemAtIndex() returns true", expectedInsertIndex, expectedInsertIndex+1)
		}

		if item != "" {
			t.Errorf("After insert at position %d expected item at position %d to be (), got (%s)", expectedInsertIndex, expectedInsertIndex+1, item)
		}
	}

	for insertValueCounter, insertValue := range []string{"six", "seven", "eight", "nine", "ten", "eleven"} {
		for i := 1; i < 5; i++ {
			stringSet[i-1] = stringSet[i]
		}
		stringSet[4] = insertValue

		q.PutItemAtEnd(insertValue)

		for i := uint(0); i < 5; i++ {
			item, hasItemAtThisIndex := q.GetItemAtIndex(i)

			if !hasItemAtThisIndex {
				t.Errorf("After insert %d expected item at position %d but GetItemAtIndex() returns false", insertValueCounter+5, i)
			}

			if item != stringSet[i] {
				t.Errorf("After insert %d expected item at position %d to be (%s), got (%s)", insertValueCounter+5, i, stringSet[i], item)
			}
		}
	}
}
