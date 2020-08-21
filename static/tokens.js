function logout() {
	document.cookie = "uuid= ; expires = Thu, 01 Jan 1970 00:00:00 GMT";
	window.location = ".";
}

function newToken(token, shop_id) {
	var description = document.getElementById("description").value;
	var exp_time = Math.floor((new Date($('#create_datetimepicker').datetimepicker('date'))).getTime() / 60000) * 60;
	var text_invalid = document.getElementById("text-invalid");
	
	var xhttp = new XMLHttpRequest();
	xhttp.onreadystatechange = function() {
		if (this.readyState == 4) {
			if (this.responseText === "ok") {
				window.location.reload(true);
			}
			else if (this.responseText === "invalid_request") {
				text_invalid.style.display = "block";
			}
			else {
				alert("Something went wrong");
			}
		}
	};
	xhttp.open("POST", "", true);
	xhttp.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
	xhttp.send(encodeURI("v=create&token=" + token + "&shop_id=" + shop_id + "&description=" + 
		description + "&exp_time=" + exp_time));
}

function deleteToken(token) {
	var xhttp = new XMLHttpRequest();
	xhttp.onreadystatechange = function() {
		if (this.readyState == 4) {
			if (this.responseText === "ok") {
				window.location.reload(true);
			}
			else {
				alert("Something went wrong");
			}
		}
	};
	xhttp.open("POST", "", true);
	xhttp.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
	xhttp.send(encodeURI("v=delete&token=" + token));
}

function editToken(token, num) {
	var shop_id = document.getElementById("shop_id" + num).value;
	var description = document.getElementById("description" + num).value;
	var exp_time = Math.floor((new Date($('#datetimepicker' + num).datetimepicker('date'))).getTime() / 60000) * 60;
	var text_invalid = document.getElementById("text-invalid" + num);

	var xhttp = new XMLHttpRequest();
	xhttp.onreadystatechange = function() {
		if (this.readyState == 4) {
			if (this.responseText === "ok") {
				window.location.reload(true);
			}
			else if (this.responseText === "invalid_request") {
				text_invalid.style.display = "block";
			}
			else {
				alert("Something went wrong");
			}
		}
	};
	xhttp.open("POST", "", true);
	xhttp.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
	xhttp.send(encodeURI("v=edit&token=" + token + "&shop_id=" + shop_id + "&description=" + 
		description + "&exp_time=" + exp_time));
}

function getRandomString() {
	return Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15);
}

function loadItems(token, num) {
	var modal_body = document.getElementById("modal_body_" + num)
	modal_body.innerHTML = "<p>Loading...</p>";

	var xhttp = new XMLHttpRequest();
	xhttp.onreadystatechange = function() {
		if (this.readyState == 4) {
			if (this.status != 200) return;
			var response = JSON.parse(xhttp.responseText);
			
			var block = "";
			for (var i = 0;i<response.length;i++) {
				var summary = response[i].item_id;
				if (response[i].color !== "") summary += ", Color: " + response[i].color;
				if (response[i].size !== "") summary += ", Size: " + response[i].size;

				var subblock = ""
				for (var j = 0;j<response[i].items.length;j++) {
					var subblock_summary = "Type: " + response[i].items[j].type + 
						", d1: " + response[i].items[j].d1 +
						", d2: " + response[i].items[j].d2 +
						", d3: " + response[i].items[j].d3 +
						", d4: " + response[i].items[j].d4 +
						", d5: " + response[i].items[j].d5; 

					id1 = getRandomString();
					id2 = getRandomString();

					subblock += "<details class=\"my-1\" id=\"" + id1 + "\"><summary>" + subblock_summary + "</summary><div id=\"" 
						+ id2 + "\" class=\"d-flex flex-row flex-wrap shadow-box rounded images-block\"></div></details>"
					$('body').on('click', '#' + id1, function(image_list, id2) {
						return function() {
							var item_container = document.getElementById(id2);
							var result = "";
							
							for (var k = 0;k<image_list.length;++k) {
								var image_id = image_list[k];
								result += "<a href=\"image/" + image_id + "\" class=\"m-1 border border-dark shadow rounded\"><img src=\"preview/" +
									image_id + "\" class=\"border border-dark shadow rounded\"></a>";
							}

							item_container.innerHTML = result;
						}
					}(response[i].items[j].image_list, id2))
				}

				block += "<details class=\"my-1\"><summary>" + summary + "</summary><div class=\"ml-4\">"
					+ subblock + "</div></details>";
			}

			if (block === "") {
				block = "<p>No items</p>";
			}

			modal_body.innerHTML = block
		}
	};
	xhttp.open("POST", "./items", true);
	xhttp.setRequestHeader('Content-type', 'application/x-www-form-urlencoded');
	xhttp.send("token=" + token);
}

function loadImages(id, image_list) {
	console.log(id, image_list)
}