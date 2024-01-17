// ==UserScript==
// @name        Melody Downloader
// @match       https://www.youtube.com/*
// @version     VERSION_PLACEHOLDER
// @run-at      document-end
// ==/UserScript==

let style = document.createElement('style');
style.innerHTML = `
#download-button {
	margin-left: 10px;
	padding: 0px 16px;
	border: none;
	border-radius: 20px;
	line-height: 36px;
	cursor: pointer;
}
`;
let head = document.getElementsByTagName('head')[0];
head.appendChild(style);

// 页面是通过 History API 管理的，切换页面并不会导致页面加载
// 所以这个函数是为了解决从 / -> /watch 的时候不会加载扩展的问题。
function isWatchPage() {
	return window.location.pathname == '/watch';
}

const apiBase = 'https://melody.home.twofei.com';

function getUrlOf(path, args, loc) {
	let url = new URL(apiBase);
	url.pathname += path ? path.substring(1) : "";
	if (typeof args == 'object') {
		let q = url.searchParams;
		Object.keys(args).forEach(key => {
			q.set(key, args[key]);
		});
	}
	if (loc) {
		url.searchParams.set('url', window.location);
	}
	return url;
}

async function handleClick() {
	let btn = createStatusButton();
	if (btn.innerText == 'Downloaded') {
		if (confirm('要删除下载？')) {
			await fetch(getUrlOf('/v1/youtube:delete', {}, true));
			await updateStatus();
		}
	} else {
		await fetch(getUrlOf('/v1/youtube:download', {}, true));
		await updateStatus();
	}
}

// 创建状态按钮
let createStatusButton = function() {
	let btn = document.getElementById('download-button');
	if (btn) return btn;
	btn = document.createElement('button');
	btn.id = 'download-button';
	let subBtn = document.querySelector('#owner #subscribe-button');
	if(!subBtn) return null;
	subBtn.parentNode.appendChild(btn);
	btn.addEventListener('click', handleClick);
	return btn;
};

async function updateStatus() {
	let statusButton = createStatusButton();
	if (!statusButton) return;

	let rsp = await fetch(getUrlOf('/v1/youtube:downloaded', {}, true));
	let status = await rsp.text();
	
	statusButton.innerText = status;
}

setInterval(async ()=> {
	if (!isWatchPage()) {return;}
	await updateStatus();
}, 5000);
