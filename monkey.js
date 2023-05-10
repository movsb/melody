function main() {
	let title = document.title;
	let channelName = document.querySelector('.watch-active-metadata #channel-name .ytd-channel-name a').innerText ;
	let likeButton = document.querySelector('.watch-active-metadata #segmented-like-button button');
	let badge = document.querySelector('.watch-active-metadata #channel-name .badge');
	let descriptionElement = document.querySelector('.watch-active-metadata #description-inline-expander span');
	let watchMetadata = document.querySelector('.watch-active-metadata #info-container');
	let hasMusicInfo = document.querySelectorAll('.watch-active-metadata ytd-video-description-music-section-renderer').length > 0;

	if (!title || !channelName || !likeButton) { return; }

	let liked = () => likeButton.getAttribute('aria-pressed') == "true";
	let badgeIsArtist = badge && badge.classList.contains('badge-style-type-verified-artist') || false;
	
	function report() {
		let data = {
			location: window.location.toString(),
			title: title,
			channelName: channelName,
			badgeIsArtist: badgeIsArtist,
			liked: liked(),
			description: descriptionElement.innerText,
			watchMetadata: watchMetadata.innerText,
			hasMusicInfo: hasMusicInfo,
		};
		
		console.log(data);
		
		fetch('https://melody.home.twofei.com/v1/youtube:like', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			// credential: 'included',
			body: JSON.stringify(data),
		});
	}
	
	window.__melody_report = report;
	
	(new MutationObserver(function (mutations) {
		mutations.forEach((mutation) => {
			if (mutation.type == 'attributes' && mutation.attributeName == 'aria-pressed') {
				// console.log(liked());
				report();
			}
		});
	}).observe(likeButton, { attributes: true }));
	
	if (liked()) {
		report();
	}
}

setInterval(()=> {
	let likeButton = document.querySelector('#segmented-like-button button');
	if (!likeButton) { console.log('没有喜欢按钮，等待中...'); }
	
	try {
		main();
	} catch (e) {
		console.log(e);
	}
}, 15000);

setInterval(async ()=> {
	let resp = await fetch('https://melody.home.twofei.com/v1/youtube:downloaded', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
		},
		// credential: 'included',
		body: JSON.stringify({
			location: window.location.toString(),
		}),
	});

	try {
		resp = await resp.json();
		let yes = resp.done;
		
		let span = document.getElementById('__melody_downloaded');
		if (span) { span.remove(); }

		span = document.createElement('span');
		span.innerText = yes ? 'Downloaded' : 'Not downloaded';
		span.style.display = 'inline-block';
		span.style.margin = 'auto 10px';
		span.id = '__melody_downloaded';
		
		let likeDislike = document.querySelector('ytd-segmented-like-dislike-button-renderer');
		likeDislike.parentNode.prepend(span);

	} catch (e) {
		console.log(e);
	}
}, 15000);
