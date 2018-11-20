let ws;
let numberPattern = /\d+/;
let searchFilter = $("#filter");
let timeout = null;

let brokenChannelContent = `
<p class="text-lg-center" style="font-size: 40px">
    Oops! Looks like channel is broken due to invalid source or parsing rule.
</p>
<button id="delete-btn" class="btn btn-outline-danger align-content-center" role="button">Delete this channel</button>
`;

function waitSocket(socket, callback) {
    setTimeout(
        function () {
            let done = false;
            if (socket) {
                if (socket.readyState === 1) {
                    callback();
                    done = true;
                }
            }
            if (!done) {
                waitSocket(socket, callback);
            }
        },
        5);
}
function deleteChannel(channelId) {
    location.href = "/deletechannel/" + channelId;
}

function isBrokenChannel() {
    let channelId = getCurrentChannel();
    let link = $("#channel-" + channelId);
    return link.hasClass("text-danger");
}

function activateChannelLink() {
    let channelId = getCurrentChannel();
    let isBroken = isBrokenChannel();
    let link = $("#channel-" + channelId);
    if (!isBroken) {
        link.addClass("active");
    } else {
        let mainContent = $("#main-content");
        mainContent.html(brokenChannelContent);
        let dltButton = $("#delete-btn");
        dltButton.attr("onclick", "deleteChannel(" + channelId + ");");
    }
}

function getCurrentChannel() {
    return parseInt(location.pathname.match(numberPattern)[0]);
}

function clearMainContent() {
    let mainContent = $("#main-content");
    mainContent.html("");
}

function updatePage() {
    clearMainContent();
    fillChannelContent();
}

function fillChannelContent() {
    let channelId = getCurrentChannel();
    let mainContent = $("#main-content");
    let filter = $("#filter");
    ws.onmessage = function (e) {
        let posts = JSON.parse(e.data);
        posts.forEach(function (post) {
            let h = document.createElement("h1");
            let link = document.createElement("a");
            let div = document.createElement("div");
            div.setAttribute("width", "100%");
            link.setAttribute("href", post.Link);
            link.innerHTML = post.Title;
            h.innerHTML = $(link).prop("outerHTML");
            div.innerHTML = post.Description;
            let hr = $(document.createElement("hr"));
            hr.addClass("hr-primary");
            mainContent.append(h);
            mainContent.append(div);
            mainContent.append(hr);
        });
    };
    ws.send(JSON.stringify({
        "Id": channelId,
        "Offset": mainContent.children().length / 3,
        "Filter": filter.val()
    }));
}

function setTypingEvent() {
    searchFilter.on("keyup", function (e) {
        clearTimeout(timeout);
        timeout = setTimeout(function () {
            updatePage();
        }, 100);
    });
}

function processPage() {
    activateChannelLink();
    if (!isBrokenChannel()) {
        fillChannelContent();
        setTypingEvent();
    }
}

$(document).ready(function () {
    ws = new WebSocket("ws://" + location.host + "/ws");
    $(window).scroll(function() {
        if ($(window).scrollTop() + $(window).height() === $(document).height()) {
            fillChannelContent();
        }
    });
    waitSocket(ws, processPage);
});
