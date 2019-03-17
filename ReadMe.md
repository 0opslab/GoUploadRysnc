相信现在稍微能说得过去的网站，后台服务器至少有俩台，所以在面对用户文件上传等功能的处理上都借助了类似nfs、fastdfs等网络文件系统来解决这类问题。单位之前一直在使用nfs，因为它足够简单有效。但是前段时间安全测发来一个测试报告，需要升级nfs才能解决。因此做了一波升级,开发运维都做确实有点累的。后来有天休假，安全侧的哥们在处理故障的时候，修改完配置后之前重启了服务器(也是醉了)，导致很多无服务上挂载的nfs都无效了，然后产生了连锁反应导致核心业务无法正常使用，然后悲催我刚和家人点的一桌菜都没上我就去救火了。

事后重新梳理了下业务流程，然后决定放弃nfs和想依赖的一些业务任务。打算用go重写一个类似的功能以方便所以人都零基础维护。因此写了GoUploadRysnc，其实原来很简单，当用户上传文件的时候有java和python做完处理校验后以http的放上上传到Go中,Go中在指定服务器上存储后并返回给Java和Python，同时利用Go的协程同步到其他服务器上。然后在这些存储文件的服务器上进行后续的业务任务。


## 使用方法
* 编译
	通过buildXXX方式即可编译相应平台的可执行文件
* 配置
	配置使用json方式，简单明了
	```json
	{
	    "addr":"0.0.0.0:9090",			//配置监地址和端口
	    "path":"c:/var/upload/wwww/",	//文件存储路径
	    "fileNameLength":11,			//文件名长度
	    "rysncAddr":[					//同步地址
	        "http://localhost:9091/rsync"
	    ]
	}
	```
* 启动
```base
UploadRysnc -conf conf/server1.conf > run.log

```

## 运行
下面是运行部分运行日志
```log
2019/03/17 10:40:00 Server is starting:0.0.0.0:9090
2019/03/17 10:40:00 Server UploadPath:c:/var/upload/wwww/
2019/03/17 10:40:00 Server Rysnc Addr:http://localhost:9091/rsync
2019/03/17 10:40:10 [::1]:49743 uploadfile [server1.conf][server1.conf] > c:/var/upload/wwww/banner/LwhSfU1nh6w.conf
2019/03/17 10:40:31 Clientrsyncfile Error http://localhost:9091/rsync c:/var/upload/wwww/banner/LwhSfU1nh6w.conf 
```