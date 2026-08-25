package queue

import (
	"math/rand/v2"

	"github.com/zyncc/ytmp/internal/models"
)

type Queue struct {
	Songs  []models.Song
	Cursor int
}

func (q *Queue) Enqueue(song models.Song) {
	q.Songs = append(q.Songs, song)
}

func (q *Queue) EnqueueAll(Songs []models.Song) {
	q.Songs = append(q.Songs, Songs...)
}

func (q *Queue) PlayNext(song models.Song) {
	if q.IsEmpty() {
		q.Enqueue(song)
		return
	}

	index := q.Cursor + 1

	q.Songs = append(q.Songs, models.Song{})
	copy(q.Songs[index+1:], q.Songs[index:])
	q.Songs[index] = song
}

func (q *Queue) Remove(index int) {
	if index < 0 || index >= len(q.Songs) {
		return
	}

	q.Songs = append(q.Songs[:index], q.Songs[index+1:]...)

	if q.Cursor > index {
		q.Cursor--
	}

	if q.Cursor >= len(q.Songs) {
		q.Cursor = len(q.Songs) - 1
	}
}

func (q *Queue) Shuffle() {
	rand.Shuffle(len(q.Songs), func(i, j int) {
		q.Songs[i], q.Songs[j] = q.Songs[j], q.Songs[i]
	})

	q.Cursor = 0
}

func (q *Queue) Next() {
	if q.Cursor < len(q.Songs)-1 {
		q.Cursor++
	}
}

func (q *Queue) Previous() {
	if q.Cursor > 0 {
		q.Cursor--
	}
}

func (q *Queue) Current() models.Song {
	if q.IsEmpty() {
		return models.Song{}
	}

	return q.Songs[q.Cursor]
}

func (q *Queue) IsEmpty() bool {
	return len(q.Songs) == 0
}

func (q *Queue) Length() int {
	return len(q.Songs)
}

func (q *Queue) Clear() {
	q.Songs = nil
	q.Cursor = 0
}
