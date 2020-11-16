vm = new Vue({
    el: "#app",
    data: {
        showHidden: false,
        breadcrumb: [],
        fileList: [
        ],
        dirName: "",
        showAlert: false,
        alertMessage: "",
        myDropzone: null,
        search: "",
    },
    created: function () {
        Dropzone.autoDiscover = false;
        var that = this;
        $(function() {
            that.myDropzone = new Dropzone("#my-awesome-dropzone", {
                paramName: "file",
                maxFilesize: 2048,
                addRemoveLinks: true,
                init: function () {
                    this.on("complete", function (file) {
                        loadFileList()
                    })
                }
            });
        })
    },
    computed: {
        computedFiles: function () {
            var that = this;
            var files;
            files = that.fileList.filter(function (f) {
                if (!that.showHidden && f.name.slice(0, 1) === '.') {
                    return false;
                }
                return true;
            });
            return files;
        }
    },
    methods: {
        clickFileOrDir: function (f, e) {
            var path = pathJoin([location.pathname, f.name]);
            if (f.is_dir) {
                loadFileOrDir(path);
                e.preventDefault()
            } else {
                window.open(path)
            }

        },
        updateBreadcrumb: function () {
            var pathName = decodeURI(location.pathname);
            var pathList = pathName.split("/");
            this.breadcrumb = [];
            if (path !== "/") {
                for(i=0; i < pathList.length; i++) {
                    if (pathList[i] !== "") {
                        var path = pathList.slice(0, i+1).join("/")
                        vm.breadcrumb.push(
                            {
                                name: pathList[i],
                                path: path
                            })
                    }
                }
            }
        },
        removeAll: function () {
            this.myDropzone.removeAllFiles();
        },
        deleteAll: function (name) {
            $.ajax({
                url: pathJoin([location.pathname, name]),
                type: "DELETE",
                success: function () {
                    loadFileList();
                },
                error: function (err) {
                    console.error(err);
                }
            })
        },
        changePath: function (path, e) {
            loadFileOrDir(path);
            e.preventDefault();
        },
        clickHidden: function () {
            this.showHidden = !this.showHidden;
        },
        initShowAlert: function () {
            this.showAlert = false;
            this.dirName = "";
        },
        newFolder: function () {
            var that = this;
            var patt = /^[0-9a-zA-Z_\.]+$/;
            if (that.dirName === "" || !patt.test(that.dirName)) {
                that.alertMessage = "Folder name should not be empty or contain illegal characters!";
                that.showAlert = true;
                return
            }
            console.log(!patt.test(that.dirName));
            $.ajax({
                url: pathJoin(["/-/mkdir", location.pathname]),
                type: "POST",
                data : {
                    name: that.dirName,
                },
                success: function () {
                    that.search = "";
                    loadFileList();
                    $('#folder-modal').modal('hide');
                },
                error: function (err) {
                    that.alertMessage = "Get error from backend server! Please check directory name '" + that.dirName + "' does not exist on server!";
                    that.showAlert = true;
                    console.error(err);
                }
            });
        },
        downloadFile: function (name) {
            window.location.href= pathJoin([location.pathname, name]) + "?download=true"
        },
        clickSearch: function () {
            loadFileList();
        }
    },
    filters: {
        formatDate: function (val) {
            var value = new Date(val);
            var year = value.getFullYear();
            var month = format(value.getMonth() + 1);
            var day = format(value.getDate());
            var hour = format(value.getHours());
            var minutes = format(value.getMinutes());
            var seconds = format(value.getSeconds());
            return year + '-' + month + '-' + day + ' ' + hour + ':' + minutes + ':' + seconds;
        },
        formatBytes: function (val) {
            var bytes = parseFloat(val);
            if (bytes < 0)
                return "-";
            else if (bytes < 1024)
                return bytes + " B";
            else if (bytes < 1048576)
                return (bytes / 1024).toFixed(0) + " KB";
            else if (bytes < 1073741824)
                return (bytes / 1048576).toFixed(1) + " MB";
            else
                return (bytes / 1073741824).toFixed(1) + " GB";
        }
    }
});

window.onpopstate = function (event) {
    vm.search = "";
    loadFileList();
};

function pathJoin(parts, sep) {
    var separator = sep || '/';
    var replace = new RegExp(separator + '{1,}', 'g');
    return parts.join(separator).replace(replace, separator);
}

function loadFileOrDir(path) {
    window.history.pushState({}, "", path);
    vm.search = "";
    loadFileList(path);
}

function loadFileList(pathname) {
    pathname = pathname || location.pathname;
    var search = vm.search !== "" ? "&search=" + vm.search : "";
    $.ajax({
        url: pathname + "?json=true" + search,
        dataType: "json",
        type: "GET",
        success: function (res) {
            res = _.sortBy(res, function (f) {
                var weight = f.is_dir ? 100 : 1;
                return -weight * f.mtime;
            })
            vm.fileList = res;
            vm.updateBreadcrumb();
        },
        error: function (err) {
            console.error(err);
        }
    })
}

function format(date) {
    if (date < 10) {
        return "0" + date;
    }
    return date;
}

function init() {
    loadFileList(location.pathname);
}

init();