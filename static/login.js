function main() {
	var login = document.getElementById('login');
	var password = document.getElementById('password');

	document.getElementById('button-login').onclick=function() {
		var xhttp = new XMLHttpRequest();
		xhttp.onreadystatechange = function() {
			if (this.readyState == 4) {
				if (this.responseText === "ok") {
					window.location = "tokens"
				}
				else if (this.responseText === "incorrect login or password"){
					login.classList.add("is-invalid");
					password.classList.add("is-invalid");
				}
			}
		};
		xhttp.open("POST", "", true);
		xhttp.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
		xhttp.send("login=" + login.value + "&password=" + password.value);
	}
}

main();