# README

该项目用于构建初始化mimo系统的可执行软件

该项目将配置文件以及资源进行压缩，并将其内嵌入可执行文件中，实现无须联网即可配置初始化mimo系统

注意，进行更改的系统最好是ubuntu20.04-live-server-amd64

其余系统不保证成功

使用前需要安装go环境且安装以下工具
```sh
sudo apt install jq -y
```

使用`pack_resources.sh`来进行静态资源的压缩并构建可执行程序

```sh
sudo ./pack_resources.sh
```

config.json里面保存了解压出来的临时目录文件以及需要复制过去的目录

例如：

```json
    {
      "src": "/tmp/mimo-output/file/boot.sh",
      "dst": "/usr/local/bin/boot.sh"
    }
```


可执行程序位于/bin

需要复制的文件位于file

如果需要增加系统需要修改的文件，将文件复制到file中，并再config里面加上需要复制到的目录

如果需要直接将spdk嵌入程序内，使用命令:
```sh
git submodule update --init
```
