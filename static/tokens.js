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
