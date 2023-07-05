import {useEffect, useState} from 'react'
import './App.css'

// Icons

function FolderIcon() {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><path d="M64 480H448c35.3 0 64-28.7 64-64V160c0-35.3-28.7-64-64-64H288c-10.1 0-19.6-4.7-25.6-12.8L243.2 57.6C231.1 41.5 212.1 32 192 32H64C28.7 32 0 60.7 0 96V416c0 35.3 28.7 64 64 64z" /></svg>
    )
}
function FileIcon() {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 384 512"><path d="M0 64C0 28.7 28.7 0 64 0H224V128c0 17.7 14.3 32 32 32H384V448c0 35.3-28.7 64-64 64H64c-35.3 0-64-28.7-64-64V64zm384 64H256V0L384 128z" /></svg>
    )
}
function DeleteIcon({callback}) {
    return (
        <svg onClick={callback} xmlns="http://www.w3.org/2000/svg" viewBox="0 0 448 512"><path d="M135.2 17.7L128 32H32C14.3 32 0 46.3 0 64S14.3 96 32 96H416c17.7 0 32-14.3 32-32s-14.3-32-32-32H320l-7.2-14.3C307.4 6.8 296.3 0 284.2 0H163.8c-12.1 0-23.2 6.8-28.6 17.7zM416 128H32L53.2 467c1.6 25.3 22.6 45 47.9 45H346.9c25.3 0 46.3-19.7 47.9-45L416 128z" /></svg>
    )
}


function MkdirBar({currentPath, updateEntry}) {
    const [name, setName] = useState("")

    const handleChange = function (e) {
        setName(e.target.value)
    }

    const mkdir = async function (e) {
        let options = {
            method: 'POST',
            body: JSON.stringify({
                path: currentPath + "/" + name
            })
        }
        let response = await fetch("/api/mkdir/", options)
        if (response.status == 200) {
            updateEntry()
        } else if (response.status == 409) {
            alert("Dir already exist")
        } else {
            alert("Sory something went wrong")
        }
    }

    return (
        <>
            <input type="text" onChange={handleChange} />
            <button onClick={mkdir}>mkdir</button>
        </>
    )
}

function UploadBar({currentPath, updateEntry}) {
    const [file, setFile] = useState(null);
    const [status, setStatus] = useState(0)

    const handleChange = (e) => {
        if (e.target.files) {
            setFile(e.target.files[0]);
        }
    };

    const upload = async function (e) {
        setStatus(1)

        let formData = new FormData();
        formData.append("file", file)
        let options = {
            method: 'POST',
            body: formData
        }
        let response = await fetch("/api/upload/?path=" + currentPath + "/" + file.name, options)
        if (response.status == 200) {
            updateEntry();
        } else if (response.status == 409) {
            alert("File already exist")
        } else {
            alert("Sory something went wrong");
        }

        setStatus(0)
        updateEntry()
    };

    if (status == 0) {
        return (
            <>
                <input type="file" onChange={handleChange} />
                <button onClick={upload}>upload</button>
            </>
        )
    } else {
        return (
            <>
                <input disabled type="file" onChange={handleChange} />
                <button disabled onClick={upload}>uploading...</button>
            </>
        )
    }

}

function DirItem({setCurrentPath, updateEntry, dir}) {
    const remove = async function (e) {
        e.stopPropagation()

        let options = {
            method: 'POST',
            body: JSON.stringify({
                path: dir.path
            })
        }
        let response = await fetch("/api/remove/", options)
        if (response.status == 200) {
            updateEntry();
        } else if (response.status == 409) {
            alert("Directory not empty")
        } else {
            alert("Sory something went wrong");
        }

        updateEntry()
    }
    return (
        <li onClick={() => setCurrentPath(dir.path)} key={dir.path} className="board-item dir">
            <FolderIcon /> <p>{dir.name}</p> <DeleteIcon callback={remove} />
        </li>
    )
}

function ParentDirItem({setCurrentPath, parentPath}) {
    return (
        <li onClick={() => setCurrentPath(parentPath)} className="board-item dir">
            <FolderIcon /> <p>..</p>
        </li>
    )
}

function FileItem({updateEntry, file}) {
    const remove = async function (e) {
        e.stopPropagation()

        let options = {
            method: 'POST',
            body: JSON.stringify({
                path: file.path
            })
        }
        let response = await fetch("/api/remove/", options)
        if (response.status == 200) {
            updateEntry();
        } else {
            alert("Sory something went wrong");
        }

        updateEntry()
    }
    const download = async function (e) {
        e.stopPropagation()
        const link = document.createElement('a');
        link.href = "/api/download/?path=" + file.path;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
    }
    return (
        <li onClick={download} key={file.path} className="board-item dir">
            <FileIcon /> <p>{file.name}</p> <DeleteIcon callback={remove} />
        </li>
    )
}

function App() {
    const [currentPath, setCurrentPath] = useState("/")
    const [parentPath, setParentPath] = useState("")

    const [dirs, setDirs] = useState([])
    const [files, setFiles] = useState([])

    const updateEntry = async function () {
        let options = {
            method: 'POST',
            body: JSON.stringify({
                path: currentPath
            })
        }
        let response = await fetch("/api/readdir/", options)
        let data = await response.json()

        if (data.dirs == undefined) {
            setDirs([])
        } else {
            setDirs(data.dirs)
        }
        if (data.files == undefined) {
            setFiles([])
        } else {
            setFiles(data.files)
        }
    }
    // const changeDir = function(path) {
    //     setCurrentPath(path)
    // }

    // update entry if change path
    useEffect(() => {
        updateEntry()
    }, [currentPath])
    // update parent path if change path
    useEffect(() => {
        if (currentPath == "/") {
            setParentPath("")
        } else {
            let path = currentPath.substring(0, currentPath.lastIndexOf("/"))
            if (path == "") {
                path = "/"
            }
            setParentPath(path)
        }
    }, [currentPath])

    return (
        <>
            <div className="board">
                <h2 className="board-name">Network storage</h2>

                <div className="board-bar">
                    <MkdirBar currentPath={currentPath} updateEntry={updateEntry} />
                </div>

                <div className="board-bar">
                    <UploadBar currentPath={currentPath} updateEntry={updateEntry} />
                </div>

                <div className="board-list">
                    <h4>path: {currentPath}</h4>
                    <ul>

                        {parentPath != "" &&
                            <ParentDirItem key=".." setCurrentPath={setCurrentPath} parentPath={parentPath} />
                        }

                        {dirs.map(dir => (
                            <DirItem key={dir.path} setCurrentPath={setCurrentPath} updateEntry={updateEntry} dir={dir} />
                        ))}

                        {files.map(file => (
                            <FileItem key={file.path} updateEntry={updateEntry} file={file} />
                        ))}
                    </ul>
                </div>
            </div>
        </>
    )
}

export default App
