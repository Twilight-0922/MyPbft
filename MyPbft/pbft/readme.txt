该项目中的一些数据库的具体含义如下：
test1.db (ToGRdigest.db)：
	table1(DataHashToGRdigest)是用来存储具体的某一个请求消息的hash到消息切片的hash映射(meta.datahash->GR.digest)
	table2(SeqToGRdigest)是用来存储消息序号到客户端请求消息切片的hash映射(sequence->GR.digest)
test2.db (ToGR.db):
	table1(GRdigestToGR)是用来存储消息切片的hash到具体消息切片的映射(GR.digest->GR)
	GRToBlock是用来存储区块的，自增存储
NodeToHash.db:
	每个节点ip是一个table(bucket桶)，存储了ip对应的具体消息hash(ip->meta.DataHash+k1(文件名)+...,自增，一个ip对应的肯定是多个datahash)
	又根据每个节点ip+isinfo创建了一个bucket，存储了meta.DataHash+k1(文件名)+...->"123"(没有任何含义)，主要是判断ip作为bucket时是否存储了meta.DataHash+k1(文件名)+...
	又创建了一个名为MissCodeChip的表名，其中k为datahash:文件名，v为缺失数量，只有当v达到一定阈值后才进行修复
RecToHash.db:
	每个meta.Reciver是一个table(bucket桶)，存储了对应的文件名+文件类型+具体消息hash(meta.Reciver->meta.DataHash,自增，同样的，一个meta.Reciver肯定对应多个datahash)
