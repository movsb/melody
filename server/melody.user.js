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

function handleClick() {
	let btn = createStatusButton();
	if (btn.innerText == 'Not Downloaded') {
		let args = new URLSearchParams;
		args.set('url', window.location);
		fetch('https://melody.home.twofei.com/v1/youtube:download?' + args.toString());
		return;
	}
	if (btn.innerText == 'Downloaded') {
		if (confirm('要删除下载？')) {
			let args = new URLSearchParams;
			args.set('url', window.location);
			fetch('https://melody.home.twofei.com/v1/youtube:delete?' + args.toString());
		}
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

async function getStatus() {
	let args = new URLSearchParams;
	args.set('url', window.location);
	let rsp = await fetch( 'https://melody.home.twofei.com/v1/youtube:downloaded?' + args.toString());
	let status = await rsp.text();
	return status;
}

setInterval(async ()=> {
	if (!isWatchPage()) {return;}
	let statusButton = createStatusButton();
	if (!statusButton) return;
	
	statusButton.innerText = await getStatus();
}, 5000);
