package redisearch

import "github.com/garyburd/redigo/redis"

// Projection
type Projection struct {
	Expression string
	Alias      string
}

func NewProjection(expression string, alias string) *Projection {
	return &Projection{
		Expression: expression,
		Alias:      alias,
	}
}

func (p Projection) Serialize() redis.Args {
	args := redis.Args{"APPLY", p.Expression, "AS", p.Alias}
	return args
}

// GroupBy
type GroupBy struct {
	Fields   []string
	Reducers []Reducer
	Paging   *Paging
}

func NewGroupBy(field string) *GroupBy {
	return &GroupBy{
		Fields:   []string{field},
		Reducers: make([]Reducer, 0),
		Paging:   nil,
	}
}

func NewGroupByFields(fields []string) *GroupBy {
	return &GroupBy{
		Fields:   fields,
		Reducers: make([]Reducer, 0),
		Paging:   nil,
	}
}

func (g *GroupBy) Reduce(reducer Reducer) *GroupBy {
	g.Reducers = append(g.Reducers, reducer)
	return g
}

func (g *GroupBy) Limit(offset int, num int) *GroupBy {
	g.Paging = NewPaging(offset, num)
	return g
}

func (g GroupBy) Serialize() redis.Args {
	ret := len(g.Fields)
	args := redis.Args{"GROUPBY", ret}.AddFlat(g.Fields)
	for _, reducer := range g.Reducers {
		args = args.AddFlat(reducer.Serialize())
	}
	if g.Paging != nil {
		args = args.AddFlat(g.Paging.serialize())
	}
	return args
}

// AggregateQuery
type AggregateQuery struct {
	Query         *Query
	AggregatePlan redis.Args
	Paging        *Paging
	Max           int
	WithSchema    bool
	Verbatim      bool
	// TODO: add cursor
	// TODO: add load fields

}

func NewAggregateQuery() *AggregateQuery {
	return &AggregateQuery{
		Query:      nil,
		Paging:     nil,
		Max:        0,
		WithSchema: false,
		Verbatim:   false,
	}
}

func (a *AggregateQuery) SetQuery(query *Query) *AggregateQuery {
	a.Query = query
	return a
}

func (a *AggregateQuery) SetWithSchema(value bool) *AggregateQuery {
	a.WithSchema = value
	return a
}

func (a *AggregateQuery) SetVerbatim(value bool) *AggregateQuery {
	a.Verbatim = value
	return a
}

func (a *AggregateQuery) SetMax(value int) *AggregateQuery {
	a.Max = value
	return a
}

//Adds a APPLY clause to the aggregate plan
func (a *AggregateQuery) Apply(expression Projection) *AggregateQuery {
	a.AggregatePlan = a.AggregatePlan.AddFlat(expression.Serialize())
	return a
}

//Sets the limit for the initial pool of results from the query.
func (a *AggregateQuery) Limit(offset int, num int) *AggregateQuery {
	a.Paging = NewPaging(offset, num)
	return a
}

//Adds a GROUPBY clause to the aggregate plan
func (a *AggregateQuery) GroupBy(group GroupBy) *AggregateQuery {
	a.AggregatePlan = a.AggregatePlan.AddFlat(group.Serialize())
	return a
}

//Adds a SORTBY clause to the aggregate plan
func (a *AggregateQuery) SortBy(SortByProperties []SortingKey) *AggregateQuery {
	nsort := len(SortByProperties)
	if nsort > 0 {
		a.AggregatePlan = a.AggregatePlan.Add("SORTBY", nsort*2)
		for _, sortby := range SortByProperties {
			a.AggregatePlan = a.AggregatePlan.AddFlat(sortby.Serialize())
		}
		if a.Max > 0 {
			a.AggregatePlan = a.AggregatePlan.Add("MAX", a.Max)
		}
	}
	return a
}

//Specify filters to filter the results using predicates relating to values in the result set.
func (a *AggregateQuery) Filter(expression string) *AggregateQuery {
	a.AggregatePlan = a.AggregatePlan.Add("FILTER", expression)
	//a.Filters = append(a.Filters, expression)
	return a
}

func (q AggregateQuery) Serialize() redis.Args {
	args := redis.Args{}
	if q.Query != nil {
		args = args.AddFlat(q.Query.serialize())
	} else {
		args = args.Add("*")
	}
	// WITHSCHEMA
	if q.WithSchema {
		args = args.Add("WITHSCHEMA")
	}
	// VERBATIM
	if q.Verbatim {
		args = args.Add("VERBATIM")
	}

	// TODO: add cursor
	// TODO: add load fields

	//Add the aggregation plan with ( GROUPBY and REDUCE | SORTBY | APPLY | FILTER ).+ clauses
	args = args.AddFlat(q.AggregatePlan)

	// LIMIT
	if q.Paging != nil {
		args = args.AddFlat(q.Paging.serialize())
	}

	return args
}
