package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"

	"github.com/boltdb/bolt"
)

type DB struct {
	DBName  string
	Handler *bolt.DB
}

// 创建表
func CreateDB(path string, tableName []byte) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		log.Println("Open error: ", err)
		return
	}
	defer db.Close()

	// 创建表1
	err = db.Update(func(tx *bolt.Tx) error {
		// 判断要创建的表是否存在
		table := tx.Bucket(tableName)
		if table == nil {
			// 创建名为talbeName的表
			_, err := tx.CreateBucket(tableName)
			if err != nil {
				log.Println("Createucket talbeName error: ", err)
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Println("CreateDB error: ", err)
		return
	}

	//// 创建表2
	//err = db.Update(func(tx *bolt.Tx) error {
	//	// 判断要创建的表是否存在
	//	table := tx.Bucket([]byte("table2"))
	//	if table == nil {
	//		// 创建名为“table2”的表
	//		_, err := tx.CreateBucket([]byte("table2"))
	//		if err != nil {
	//			log.Fatal(err)
	//		}
	//	}
	//	return nil
	//})
	//if err != nil {
	//	log.Fatal(err)
	//}
}

// 更新表
func Update(path string, tableName []byte, key []byte, value []byte) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		log.Println("Open error: ", err)
		return
	}
	defer db.Close()

	// 创建表tableName
	err = db.Update(func(tx *bolt.Tx) error {
		//如果表tableName不存在则创建表
		table, err := tx.CreateBucketIfNotExists(tableName)
		//向表中写入数据
		err = table.Put(key, value)
		if err != nil {
			log.Println("Writer error: ", err)
			return err
		}
		return nil
	})

	//err = db.Update(func(tx *bolt.Tx) error {
	//	// 指定写入数据的表名
	//	table := tx.Bucket(tableName)
	//
	//	// 向表中写入数据
	//	if table != nil {
	//		err := table.Put(key, value)
	//		if err != nil {
	//			log.Println("Writer error: ", err)
	//			return err
	//		}
	//	}
	//	return nil
	//})

	// 更新数据库失败
	if err != nil {
		log.Println("Update error: ", err)
		return
	}
}

// 查询表
func Search(path string, tableName []byte, key []byte) []byte {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		log.Println("Open error: ", err)
		return nil
	}
	defer db.Close()
	var data2 []byte
	err = db.View(func(tx *bolt.Tx) error {
		// 指定查询表
		table := tx.Bucket(tableName)
		// 从表中查询数据
		if table != nil {
			data := table.Get(key)
			// fmt.Printf("%s\n", data) // 根据需要编写其他执行逻辑
			data2 = append(data2, data...) //将data切片的数据复制到另一个切片里
		} else {
			return errors.New("没有找到表名为：" + string(tableName) + "的表")
		}
		return nil
	})
	//查询失败
	if err != nil {
		log.Println("View error：", err)
		return nil
	}

	return data2
}

// 返回v的8字节大端表示，uint64转[]byte
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

// 自增表实现映射
func Insert(path string, tableName string, value []byte) error {
	//首先查看是否表名+isinfo中存在该值
	if Search(path, []byte(tableName+"isinfo"), value) != nil { //如果在表中找到了该值，则去重
		return nil
	}

	var err error
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return err
	}
	//defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		table := tx.Bucket([]byte(tableName))

		if table != nil {
			id, err := table.NextSequence()
			if err != nil {
				fmt.Println("table.NextSequence() err is ", err)
			}
			// fmt.Println(itob(id))
			err = table.Put(itob(id), value)
			if err != nil {
				fmt.Println("table.Put() err is ", err)
			}
		} else {
			// 表不存在，先创表，再写入数据
			_, err := tx.CreateBucket([]byte(tableName))
			if err != nil {
				fmt.Println("tx.CreateBucket() err is ", err)
			}
			table := tx.Bucket([]byte(tableName))
			id, err := table.NextSequence()
			if err != nil {
				fmt.Println("table.NextSequence() err is ", err)
			}
			err = table.Put(itob(id), value)
			if err != nil {
				fmt.Println("table.Put() err is ", err)
			}
		}
		return nil
	})

	db.Close()
	// 插入数据失败
	if err != nil {
		fmt.Println("Insert data err!")
	} else {
		//当插入数据成功时 更新表名+isinfo中的值
		Update(path, []byte(tableName+"isinfo"), value, []byte("123"))
	}
	return err
}

// 遍历查找--通过节点id查找datahash
func Search2(path string, tableName string) (error, []map[uint64]string) {
	var err error
	pair := make(map[uint64]string)
	var maps []map[uint64]string
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return err, nil
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		// 指定查询表
		table := tx.Bucket([]byte(tableName))

		i := table.Cursor() // 表指针

		for k, v := i.First(); k != nil; k, v = i.Next() {
			// key := binary.LittleEndian.Uint64(k)
			key := binary.BigEndian.Uint64(k)
			value := string(v)
			pair[key] = value
		}
		maps = append(maps, pair)
		return nil
	})

	if err != nil {
		fmt.Println("Search data fail!")
	}
	return err, maps
}
