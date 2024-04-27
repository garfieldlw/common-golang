package mongo

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
	"strings"
	"sync"
	"time"
)

const (
	MaxConnection     = 10
	InitialConnection = 1
)

var mu sync.RWMutex

type mongoData struct {
	Client *mongo.Client
	pos    int
	flag   bool
}

type ClientPool struct {
	clientList [MaxConnection]mongoData
	size       int
}

var cp ClientPool

// initial the connection to the pool
func init() {
	for size := 0; size < InitialConnection || size < MaxConnection; size++ {
		err := cp.allocateToPool(size)
		if err != nil {
			return
		}

		cp.clientList[size].flag = false
	}
}

func connect() (*mongo.Client, error) {
	conf := GetMongoConfig()
	if conf == nil {
		return nil, errors.New("get mongo config fail")
	}

	url := fmt.Sprintf("mongodb://%v:%v@%v:%v/?authSource=admin", conf.User, conf.Password, conf.Host, conf.Port)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(url))
	if err != nil {
		return nil, err
	}

	return client, nil
}

func disconnect(client *mongo.Client) error {
	return client.Disconnect(context.Background())
}

func (cp *ClientPool) allocateToPool(pos int) (err error) {
	cp.clientList[pos].Client, err = connect()
	if err != nil {
		return err
	}

	cp.clientList[pos].flag = true
	cp.clientList[pos].pos = pos
	return nil
}

func (cp *ClientPool) getToPool(pos int) {
	cp.clientList[pos].flag = true
}

func (cp *ClientPool) putBackPool(pos int) {
	cp.clientList[pos].flag = false
}

func GetClient() (*mongoData, error) {
	mu.RLock()
	for i := 1; i < cp.size; i++ {
		if cp.clientList[i].flag == false {
			mu.RUnlock()
			return &cp.clientList[i], nil
		}
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()
	if cp.size < MaxConnection {
		err := cp.allocateToPool(cp.size)
		if err != nil {
			return nil, err
		}

		pos := cp.size
		cp.size++
		return &cp.clientList[pos], nil
	} else {
		return nil, errors.New("pooling is fulled")
	}
}

func ReleaseClient(c *mongoData) {
	mu.Lock()
	defer mu.Unlock()
	cp.putBackPool(c.pos)
}

func ErrorIsNoDocuments(err error) bool {
	return errors.Is(
		err,
		mongo.ErrNoDocuments,
	)
}

type Mongo struct {
	whereMap       bson.D
	selectMap      []string
	limit          int
	skip           int
	collectionName string
	database       string
	group          []string
}

func (m *Mongo) InsertOne(ctx context.Context, db *mongoData, data interface{}) (*mongo.InsertOneResult, error) {
	colName := m.getCollectionName(data)
	if len(m.selectMap) > 0 {
		data = m.filterColumn(data)
	}
	res, err := db.Client.Database(m.database).Collection(colName).InsertOne(ctx, data)
	return res, err
}

func (m *Mongo) InsertMany(ctx context.Context, db *mongoData, data []interface{}) (*mongo.InsertManyResult, error) {
	rt := reflect.TypeOf(data)
	rtValue := reflect.ValueOf(data)
	if rt.Kind() == reflect.Slice && rtValue.Len() > 0 {
		var arr []interface{}
		colName := m.getCollectionName(data[0])
		if len(m.selectMap) > 0 {
			for i := 0; i < rtValue.Len(); i++ {
				filterData := m.filterColumn(rtValue.Index(i).Interface())
				arr = append(arr, filterData)
			}
			data = arr
		}
		many, err := db.Client.Database(m.database).Collection(colName).InsertMany(ctx, data)
		if err != nil {
			return nil, err
		}
		return many, nil
	}
	return nil, errors.New("data type error")
}

func (m *Mongo) Limit(limit int) *Mongo {
	m.limit = limit
	return m
}

func (m *Mongo) Skip(skip int) *Mongo {
	m.skip = skip
	return m
}

func (m *Mongo) Find(ctx context.Context, db *mongoData, data interface{}) {
	var selectOp, limitOp, skipOp *options.FindOptions
	if len(m.whereMap) == 0 {
		m.whereMap = bson.D{}
	}
	colName := m.getCollectionName(data)
	var d bson.D
	if len(m.selectMap) > 0 {
		for _, value := range m.selectMap {
			d = append(d, bson.E{Key: value, Value: 1})
		}
	}
	optionsObj := options.Find()
	selectOp = optionsObj.SetProjection(d)
	if m.limit > 0 {
		limitOp = optionsObj.SetLimit(int64(m.limit))
	}
	if m.skip > 0 {
		skipOp = optionsObj.SetSkip(int64(m.skip))
	}
	cursor, err := db.Client.Database(m.database).Collection(colName).Find(ctx, m.whereMap, selectOp, limitOp, skipOp)
	err = cursor.All(ctx, data)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func (m *Mongo) FindOne(ctx context.Context, db *mongoData, data interface{}, options ...*options.FindOneOptions) {
	if len(m.whereMap) == 0 {
		m.whereMap = bson.D{}
	}
	colName := m.getCollectionName(data)
	err := db.Client.Database(m.database).Collection(colName).FindOne(ctx, m.whereMap, options...).Decode(data)
	if err != nil {
		return
	}
}

func (m *Mongo) Where(where bson.D) *Mongo {
	m.whereMap = where
	return m
}

func (m *Mongo) Select(fields ...string) *Mongo {
	for _, v := range fields {
		m.selectMap = append(m.selectMap, v)
	}
	return m
}

func (m *Mongo) SetCollectionName(name string) *Mongo {
	m.collectionName = name
	return m
}

func (m *Mongo) SetDatabase(name string) *Mongo {
	m.database = name
	return m
}

func (m *Mongo) Group(group ...string) *Mongo {
	for _, v := range group {
		m.group = append(m.group, v)
	}
	return m
}

func (m *Mongo) Count(ctx context.Context, db *mongoData, data interface{}) {
	var pipeline bson.A
	colName := m.getCollectionName(data)
	if len(m.whereMap) > 0 {
		pipeline = append(pipeline, m.whereMap)
	}
	var groupMap = bson.M{}
	if len(m.group) > 0 {
		var groupItem = bson.M{}
		for _, value := range m.group {
			groupItem[value] = fmt.Sprintf("$%s", value)
		}
		groupMap["$group"] = bson.M{"_id": groupItem, "num": bson.M{"$sum": 1}}
		pipeline = append(pipeline, groupMap)
	}
	cursor, err := db.Client.Database(m.database).Collection(colName).Aggregate(ctx, pipeline)
	if err != nil {
		return
	}
	err = cursor.All(ctx, data)
	if err != nil {
		return
	}
}

func (m *Mongo) Aggregate(ctx context.Context, db *mongoData, data interface{}) []bson.M {
	var res []bson.M
	colName := m.getCollectionName(data)
	if len(m.selectMap) > 0 {
		data = m.filterColumn(data)
	}
	cursor, err := db.Client.Database(m.database).Collection(colName).Aggregate(ctx, data)
	fmt.Println(cursor)
	err = cursor.All(ctx, &res)
	if err != nil {
		fmt.Println(err.Error())
	}
	return res
}

func (m *Mongo) getCollectionName(collection interface{}) string {
	if m.collectionName != "" {
		return m.collectionName
	}
	colType := reflect.TypeOf(collection)
	if colType.Kind() == reflect.Ptr {
		colType = colType.Elem()
	}
	var newReflect reflect.Value
	if colType.Kind() == reflect.Slice {
		newReflect = reflect.New(colType.Elem())
	} else {
		newReflect = reflect.New(colType)
	}
	method := newReflect.MethodByName("CollectionName")
	if method.IsValid() {
		res := method.Call(nil)
		for _, v := range res {
			return v.String()
		}
	}
	panic("invalid collection")
}

func (m *Mongo) filterColumn(data interface{}) bson.D {
	var d bson.D
	rt := reflect.TypeOf(data)
	rtValue := reflect.ValueOf(data)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
		rtValue = rtValue.Elem()
	}
	if rt.Kind() == reflect.Struct {
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			key := strings.Split(f.Tag.Get("bson"), ",")[0]
			if key == "" {
				key = f.Name
			}
			if ok, err := m.inArray(f.Name, m.selectMap); ok && err == nil {
				d = append(d, bson.E{Key: key, Value: rtValue.Field(i).Interface()})
			}
		}
	}
	return d
}

func (m *Mongo) inArray(item interface{}, arr interface{}) (bool, error) {
	arrValue := reflect.ValueOf(arr)
	for i := 0; i < arrValue.Len(); i++ {
		if reflect.ValueOf(arrValue.Index(i).Interface()).Kind() != reflect.ValueOf(item).Kind() {
			return false, errors.New("invalid type")
		}
		if reflect.ValueOf(arrValue.Index(i).Interface()).Interface() == reflect.ValueOf(item).Interface() {
			return true, nil
		}
	}
	return false, nil
}

type ConfigItem struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

func GetMongoConfig() *ConfigItem {
	return &ConfigItem{
		Host:     "",
		Port:     "",
		User:     "",
		Password: "",
	}
}
