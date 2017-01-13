package bsdiff

import (
	"bytes"
	"fmt"
	"runtime"
	"time"

	"github.com/itchio/wharf/state"
)

// Ternary-Split Quicksort, cf. http://www.larsson.dogma.net/ssrev-tr.pdf
// Does: [  < x  ][  = x  ][  > x  ]
// V is read-only, V2 is written to — this allows parallelism.
func split(I, V, V2 []int32, start, length, h int32) {
	var i, j, k, x, jj, kk int32

	// selection sort, for small buckets (don't split any further)
	if length < 16 {
		for k = start; k < start+length; k += j {
			// the subarray [start:k] is already sorted
			j = 1
			// using the doubling technique from Karp, Miller, and Rosenberg,
			// V[I[i]+h] is our sorting key. See section 2.1 of
			// http://www.larsson.dogma.net/ssrev-tr.pdf
			x = V[I[k]+h]
			for i = 1; k+i < start+length; i++ {
				if V[I[k+i]+h] < x {
					// found a smaller value, x is the new smallest value
					x = V[I[k+i]+h]
					j = 0
				}
				if V[I[k+i]+h] == x {
					// since x is the smallest value we've seen so far, swap
					// the (k+i)th element next to it
					I[k+i], I[k+j] = I[k+j], I[k+i]
					// j is the number of values equal to the smallest value
					j++
				}
			}

			// at this point, x is the smallest value of the right part
			// of the array (the one we're still sorting) — and all values
			// equal to X are
			for i = 0; i < j; i++ {
				// commit group number for all values == x.
				// k + j -1 is a constant, they're all in the same group
				// (if j > 1, the group is not sorted yet).
				V2[I[k+i]] = k + j - 1
			}
			if j == 1 {
				// if there was only one value = x, it's a sorted group, mark
				// it as such in I (see faster suffix sorting)
				I[k] = -1
			}
		}
		return
	}

	// find pivot
	x = V[I[start+length/2]+h]
	jj = 0
	kk = 0
	for i = start; i < start+length; i++ {
		if V[I[i]+h] < x {
			// size of < bucket
			jj++
		}
		if V[I[i]+h] == x {
			// size of = bucket
			kk++
		}
	}
	// last index of < bucket
	jj += start
	// last index of = bucket
	kk += jj

	i = start
	j = 0
	k = 0
	// i goes from start of sorted array to end of < bucket
	for i < jj {
		if V[I[i]+h] < x {
			// ith element belongs in < bucket
			i++
		} else if V[I[i]+h] == x {
			// swap with = bucket
			I[i], I[jj+j] = I[jj+j], I[i]
			// j is our current position in the = bucket
			j++
		} else {
			// swap with > bucket
			I[i], I[kk+k] = I[kk+k], I[i]
			// k is our current position in the > bucket
			k++
		}
	}

	// at this point, the < bucket contains all < elements
	// but the = bucket might contain > elements, and vice versa
	for jj+j < kk {
		if V[I[jj+j]+h] == x {
			// (jj+j)th elements really belongs in =
			j++
		} else {
			// swap with > bucket
			I[jj+j], I[kk+k] = I[kk+k], I[jj+j]
			k++
		}
	}

	// at this point, the < bucket contains
	// all values < x, unsorted. same goes
	// for = and > buckets

	if jj > start {
		// < bucket is not empty, sort it
		split(I, V, V2, start, jj-start, h)
	}

	for i = 0; i < kk-jj; i++ {
		// commit ordering of = bucket
		// note that `kk - 1` is constant: all = elements have
		// the same group number, see Definition 3
		// in http://www.larsson.dogma.net/ssrev-tr.pdf
		V2[I[jj+i]] = kk - 1
	}
	if jj == kk-1 {
		// if = bucket is empty, that means we've
		// sorted the group (no need for further subsorts)
		I[jj] = -1
	}

	if start+length > kk {
		// > bucket is not empty, sort it
		split(I, V, V2, kk, start+length-kk, h)
	}
}

// important: keep in sync with split
func split64(I, V, V2 []int64, start, length, h int64) {
	var i, j, k, x, jj, kk int64

	// selection sort, for small buckets (don't split any further)
	if length < 16 {
		for k = start; k < start+length; k += j {
			// the subarray [start:k] is already sorted
			j = 1
			// using the doubling technique from Karp, Miller, and Rosenberg,
			// V[I[i]+h] is our sorting key. See section 2.1 of
			// http://www.larsson.dogma.net/ssrev-tr.pdf
			x = V[I[k]+h]
			for i = 1; k+i < start+length; i++ {
				if V[I[k+i]+h] < x {
					// found a smaller value, x is the new smallest value
					x = V[I[k+i]+h]
					j = 0
				}
				if V[I[k+i]+h] == x {
					// since x is the smallest value we've seen so far, swap
					// the (k+i)th element next to it
					I[k+i], I[k+j] = I[k+j], I[k+i]
					// j is the number of values equal to the smallest value
					j++
				}
			}

			// at this point, x is the smallest value of the right part
			// of the array (the one we're still sorting) — and all values
			// equal to X are
			for i = 0; i < j; i++ {
				// commit group number for all values == x.
				// k + j -1 is a constant, they're all in the same group
				// (if j > 1, the group is not sorted yet).
				V2[I[k+i]] = k + j - 1
			}
			if j == 1 {
				// if there was only one value = x, it's a sorted group, mark
				// it as such in I (see faster suffix sorting)
				I[k] = -1
			}
		}
		return
	}

	// find pivot
	x = V[I[start+length/2]+h]
	jj = 0
	kk = 0
	for i = start; i < start+length; i++ {
		if V[I[i]+h] < x {
			// size of < bucket
			jj++
		}
		if V[I[i]+h] == x {
			// size of = bucket
			kk++
		}
	}
	// last index of < bucket
	jj += start
	// last index of = bucket
	kk += jj

	i = start
	j = 0
	k = 0
	// i goes from start of sorted array to end of < bucket
	for i < jj {
		if V[I[i]+h] < x {
			// ith element belongs in < bucket
			i++
		} else if V[I[i]+h] == x {
			// swap with = bucket
			I[i], I[jj+j] = I[jj+j], I[i]
			// j is our current position in the = bucket
			j++
		} else {
			// swap with > bucket
			I[i], I[kk+k] = I[kk+k], I[i]
			// k is our current position in the > bucket
			k++
		}
	}

	// at this point, the < bucket contains all < elements
	// but the = bucket might contain > elements, and vice versa
	for jj+j < kk {
		if V[I[jj+j]+h] == x {
			// (jj+j)th elements really belongs in =
			j++
		} else {
			// swap with > bucket
			I[jj+j], I[kk+k] = I[kk+k], I[jj+j]
			k++
		}
	}

	// at this point, the < bucket contains
	// all values < x, unsorted. same goes
	// for = and > buckets

	if jj > start {
		// < bucket is not empty, sort it
		split64(I, V, V2, start, jj-start, h)
	}

	for i = 0; i < kk-jj; i++ {
		// commit ordering of = bucket
		// note that `kk - 1` is constant: all = elements have
		// the same group number, see Definition 3
		// in http://www.larsson.dogma.net/ssrev-tr.pdf
		V2[I[jj+i]] = kk - 1
	}
	if jj == kk-1 {
		// if = bucket is empty, that means we've
		// sorted the group (no need for further subsorts)
		I[jj] = -1
	}

	if start+length > kk {
		// > bucket is not empty, sort it
		split64(I, V, V2, kk, start+length-kk, h)
	}
}

type mark struct {
	index int32
	value int32
}

type mark64 struct {
	index int64
	value int64
}

type sortTask struct {
	start  int32
	length int32
	h      int32
}

type sortTask64 struct {
	start  int64
	length int64
	h      int64
}

// Faster Suffix Sorting, see: http://www.larsson.dogma.net/ssrev-tr.pdf
// Output `I` is a sorted suffix array.
// TODO: implement parallel sorting as a faster alternative for high-RAM environments
// see http://www.zbh.uni-hamburg.de/pubs/pdf/FutAluKur2001.pdf
func qsufsort(obuf []byte, ctx *DiffContext, consumer *state.Consumer) []int32 {
	parallel := ctx.SuffixSortConcurrency != 0
	numWorkers := ctx.SuffixSortConcurrency
	if numWorkers < 1 {
		numWorkers += runtime.NumCPU()
	}

	var buckets [256]int32
	var i, h int32
	var obuflen = int32(len(obuf))

	I := make([]int32, obuflen+1)
	V := make([]int32, obuflen+1)

	for _, c := range obuf {
		buckets[c]++
	}
	for i = 1; i < 256; i++ {
		buckets[i] += buckets[i-1]
	}
	copy(buckets[1:], buckets[:])
	buckets[0] = 0

	for i, c := range obuf {
		buckets[c]++
		I[buckets[c]] = int32(i)
	}

	I[0] = obuflen
	for i, c := range obuf {
		V[i] = buckets[c]
	}

	V[obuflen] = 0
	for i = 1; i < 256; i++ {
		if buckets[i] == buckets[i-1]+1 {
			I[buckets[i]] = -1
		}
	}
	I[0] = -1

	const progressInterval = 64 * 1024

	var V2 []int32
	var marks []mark

	if parallel {
		consumer.Debugf("parallel suffix sorting (%d workers)", numWorkers)
		V2 = append([]int32{}, V...)
		marks = make([]mark, 0)
	} else {
		consumer.Debugf("single-core suffix sorting")
	}

	// we buffer the tasks channel so that we can queue workloads (and
	// combine sorted groups) faster than workers can handle them: this helps throughput.
	// picking a value too small would lower core utilization.
	// picking a value too large would add overhead, negating the benefits.
	taskBufferSize := numWorkers * 4

	done := make(chan bool)
	var copyStart time.Time
	var copyDuration time.Duration

	for h = 1; I[0] != -(obuflen + 1); h += h {
		// in practice, h < 32, so this is a calculated waste of memory
		tasks := make(chan sortTask, taskBufferSize)

		if parallel {
			// in parallel mode, fan-out sorting tasks to a few workers
			for i := 0; i < numWorkers; i++ {
				go func() {
					for task := range tasks {
						// see split's definition for why V and V2 are necessary
						split(I, V, V2, task.start, task.length, task.h)
					}
					done <- true
				}()
			}

			// keep track of combined groups we found while scanning I
			marks = marks[:0]
		}

		consumer.ProgressLabel(fmt.Sprintf("Suffix sorting (%d-order)", h))

		// used to combine adjacent sorted groups into a single, bigger sorted group
		// eventually we'll be left with a single sorted group of size len(obuf)+1
		var n int32

		// total number of suffixes sorted at the end of this pass
		var nTotal int32

		// last index at which we emitted progress info
		var lastI int32

		for i = 0; i < obuflen+1; {
			if i-lastI > progressInterval {
				// calling Progress on every iteration woudl slow down diff significantly
				progress := float64(i) / float64(obuflen)
				consumer.Progress(progress)
				lastI = i
			}

			if I[i] < 0 {
				// found a combined-sorted group
				// n accumulates adjacent combined-sorted groups
				n -= I[i]

				// count towards total number of suffixes sorted
				nTotal -= I[i]

				// skip over it, since it's already sorted
				i -= I[i]
			} else {
				if n != 0 {
					// before we encountered this group, we had "-n" sorted suffixes
					// (potentially from different groups), merge them into a single group
					if parallel {
						// if working in parallel, only mark after tasks are done, otherwise
						// it messes with indices the quicksort is relying on
						marks = append(marks, mark{index: i - n, value: -n})
					} else {
						// if working sequentially, we can mark them immediately.
						I[i-n] = -n
					}
				}

				// retrieve size of group to sort (g - f + 1), where g is the group number
				// and f is the index of the start of the group (i, here)
				n = V[I[i]] + 1 - i

				// only hand out sorts to other cores if:
				//   - we're doing a parallel suffix sort,
				//   - the array to sort is big enough
				// otherwise, the overhead cancels the performance gains.
				// this means not all cores will always be maxed out
				// (especially in later passes), but we'll still complete sooner
				if parallel && n > 128 {
					tasks <- sortTask{
						start:  i,
						length: n,
						h:      h,
					}
				} else {
					if parallel {
						// other groups might be sorted in parallel, still need to use V and V2
						split(I, V, V2, i, n, h)
					} else {
						// no need for V2 in sequential mode, only one core ever reads/write to V
						split(I, V, V, i, n, h)
					}
				}

				// advance over entire group
				i += n
				// reset "combined sorted group" length accumulator
				n = 0
			}
		}

		if parallel {
			// this will break out of the "for-range" of the workers when
			// the channel's buffer is empty
			close(tasks)
			for i := 0; i < numWorkers; i++ {
				// workers cannot err, only panic, we're just looking for completion here
				<-done
			}

			// we can now safely mark groups as sorted
			for _, mark := range marks {
				// consumer.Debugf("Setting I[%d] to %d", I[i-n], -n)
				I[mark.index] = mark.value
			}
		}

		if n != 0 {
			// eventually, this will write I[0] = -(len(obuf) + 1), when
			// all suffixes are sorted. until then, it'll catch the last combined
			// sorted group
			I[i-n] = -n
		}

		// consumer.Debugf("%d/%d was already done (%.2f%%)", doneI, (obuflen + 1),
		// 	100.0*float64(doneI)/float64(obuflen+1))

		if parallel {
			if ctx.MeasureParallelOverhead {
				copyStart = time.Now()
				copy(V, V2)
				copyDuration += time.Since(copyStart)
			} else {
				copy(V, V2)
			}
		}
	}

	if parallel && ctx.MeasureParallelOverhead {
		consumer.Debugf("Parallel copy overhead: %s", copyDuration)
	}

	// at this point, V[i] contains the group number of the ith suffix:
	// all groups are now of size 1, so V[i] is the final position of the
	// suffix in the list of indices of sorted suffixes. Commit it to I,
	// our result.
	for i = 0; i < obuflen+1; i++ {
		I[V[i]] = i
	}
	return I
}

// important: keep in sync with qsufsort
func qsufsort64(obuf []byte, ctx *DiffContext, consumer *state.Consumer) []int64 {
	parallel := ctx.SuffixSortConcurrency != 0
	numWorkers := ctx.SuffixSortConcurrency
	if numWorkers < 1 {
		numWorkers += runtime.NumCPU()
	}

	var buckets [256]int64
	var i, h int64
	var obuflen = int64(len(obuf))

	I := make([]int64, obuflen+1)
	V := make([]int64, obuflen+1)

	for _, c := range obuf {
		buckets[c]++
	}
	for i = 1; i < 256; i++ {
		buckets[i] += buckets[i-1]
	}
	copy(buckets[1:], buckets[:])
	buckets[0] = 0

	for i, c := range obuf {
		buckets[c]++
		I[buckets[c]] = int64(i)
	}

	I[0] = obuflen
	for i, c := range obuf {
		V[i] = buckets[c]
	}

	V[obuflen] = 0
	for i = 1; i < 256; i++ {
		if buckets[i] == buckets[i-1]+1 {
			I[buckets[i]] = -1
		}
	}
	I[0] = -1

	const progressInterval = 64 * 1024

	var V2 []int64
	var marks []mark64

	if parallel {
		consumer.Debugf("parallel suffix sorting (%d workers)", numWorkers)
		V2 = append([]int64{}, V...)
		marks = make([]mark64, 0)
	} else {
		consumer.Debugf("single-core suffix sorting")
	}

	// we buffer the tasks channel so that we can queue workloads (and
	// combine sorted groups) faster than workers can handle them: this helps throughput.
	// picking a value too small would lower core utilization.
	// picking a value too large would add overhead, negating the benefits.
	taskBufferSize := numWorkers * 4

	done := make(chan bool)
	var copyStart time.Time
	var copyDuration time.Duration

	for h = 1; I[0] != -(obuflen + 1); h += h {
		// in practice, h < 32, so this is a calculated waste of memory
		tasks := make(chan sortTask64, taskBufferSize)

		if parallel {
			// in parallel mode, fan-out sorting tasks to a few workers
			for i := 0; i < numWorkers; i++ {
				go func() {
					for task := range tasks {
						// see split's definition for why V and V2 are necessary
						split64(I, V, V2, task.start, task.length, task.h)
					}
					done <- true
				}()
			}

			// keep track of combined groups we found while scanning I
			marks = marks[:0]
		}

		consumer.ProgressLabel(fmt.Sprintf("Suffix sorting (%d-order)", h))

		// used to combine adjacent sorted groups into a single, bigger sorted group
		// eventually we'll be left with a single sorted group of size len(obuf)+1
		var n int64

		// total number of suffixes sorted at the end of this pass
		var nTotal int64

		// last index at which we emitted progress info
		var lastI int64

		for i = 0; i < obuflen+1; {
			if i-lastI > progressInterval {
				// calling Progress on every iteration woudl slow down diff significantly
				progress := float64(i) / float64(obuflen)
				consumer.Progress(progress)
				lastI = i
			}

			if I[i] < 0 {
				// found a combined-sorted group
				// n accumulates adjacent combined-sorted groups
				n -= I[i]

				// count towards total number of suffixes sorted
				nTotal -= I[i]

				// skip over it, since it's already sorted
				i -= I[i]
			} else {
				if n != 0 {
					// before we encountered this group, we had "-n" sorted suffixes
					// (potentially from different groups), merge them into a single group
					if parallel {
						// if working in parallel, only mark after tasks are done, otherwise
						// it messes with indices the quicksort is relying on
						marks = append(marks, mark64{index: i - n, value: -n})
					} else {
						// if working sequentially, we can mark them immediately.
						I[i-n] = -n
					}
				}

				// retrieve size of group to sort (g - f + 1), where g is the group number
				// and f is the index of the start of the group (i, here)
				n = V[I[i]] + 1 - i

				// only hand out sorts to other cores if:
				//   - we're doing a parallel suffix sort,
				//   - the array to sort is big enough
				// otherwise, the overhead cancels the performance gains.
				// this means not all cores will always be maxed out
				// (especially in later passes), but we'll still complete sooner
				if parallel && n > 128 {
					tasks <- sortTask64{
						start:  i,
						length: n,
						h:      h,
					}
				} else {
					if parallel {
						// other groups might be sorted in parallel, still need to use V and V2
						split64(I, V, V2, i, n, h)
					} else {
						// no need for V2 in sequential mode, only one core ever reads/write to V
						split64(I, V, V, i, n, h)
					}
				}

				// advance over entire group
				i += n
				// reset "combined sorted group" length accumulator
				n = 0
			}
		}

		if parallel {
			// this will break out of the "for-range" of the workers when
			// the channel's buffer is empty
			close(tasks)
			for i := 0; i < numWorkers; i++ {
				// workers cannot err, only panic, we're just looking for completion here
				<-done
			}

			// we can now safely mark groups as sorted
			for _, mark := range marks {
				// consumer.Debugf("Setting I[%d] to %d", I[i-n], -n)
				I[mark.index] = mark.value
			}
		}

		if n != 0 {
			// eventually, this will write I[0] = -(len(obuf) + 1), when
			// all suffixes are sorted. until then, it'll catch the last combined
			// sorted group
			I[i-n] = -n
		}

		// consumer.Debugf("%d/%d was already done (%.2f%%)", doneI, (obuflen + 1),
		// 	100.0*float64(doneI)/float64(obuflen+1))

		if parallel {
			if ctx.MeasureParallelOverhead {
				copyStart = time.Now()
				copy(V, V2)
				copyDuration += time.Since(copyStart)
			} else {
				copy(V, V2)
			}
		}
	}

	if parallel && ctx.MeasureParallelOverhead {
		consumer.Debugf("Parallel copy overhead: %s", copyDuration)
	}

	// at this point, V[i] contains the group number of the ith suffix:
	// all groups are now of size 1, so V[i] is the final position of the
	// suffix in the list of indices of sorted suffixes. Commit it to I,
	// our result.
	for i = 0; i < obuflen+1; i++ {
		I[V[i]] = i
	}
	return I
}

// Returns the number of bytes common to a and b
func matchlen(a, b []byte) (i int32) {
	alen := int32(len(a))
	blen := int32(len(b))
	for i < alen && i < blen && a[i] == b[i] {
		i++
	}
	return i
}

// important: keep in sync with matchlen
func matchlen64(a, b []byte) (i int64) {
	alen := int64(len(a))
	blen := int64(len(b))
	for i < alen && i < blen && a[i] == b[i] {
		i++
	}
	return i
}

// Do a binary search in our (sorted) suffix array to find the closest suffix
func search(I []int32, obuf, nbuf []byte, st, en int32) (pos, n int32) {
	if en-st < 2 {
		x := matchlen(obuf[I[st]:], nbuf)
		y := matchlen(obuf[I[en]:], nbuf)

		if x > y {
			return I[st], x
		}
		return I[en], y
	}

	x := st + (en-st)/2
	if bytes.Compare(obuf[I[x]:], nbuf) < 0 {
		return search(I, obuf, nbuf, x, en)
	}
	return search(I, obuf, nbuf, st, x)
}

func search64(I []int64, obuf, nbuf []byte, st, en int64) (pos, n int64) {
	if en-st < 2 {
		x := matchlen64(obuf[I[st]:], nbuf)
		y := matchlen64(obuf[I[en]:], nbuf)

		if x > y {
			return I[st], x
		}
		return I[en], y
	}

	x := st + (en-st)/2
	if bytes.Compare(obuf[I[x]:], nbuf) < 0 {
		return search64(I, obuf, nbuf, x, en)
	}
	return search64(I, obuf, nbuf, st, x)
}
