export {};

const form = document.querySelector<HTMLFormElement>("#registration")!;
const username = document.querySelector<HTMLInputElement>("#username")!;
const password = document.querySelector<HTMLInputElement>("#password")!;
const confirmation = document.querySelector<HTMLInputElement>("#confirmPassword")!;
const submit = document.querySelector<HTMLButtonElement>("#submit")!;
const status = document.querySelector<HTMLParagraphElement>("#status")!;
let pending = false;

function show(message: string, kind: "error" | "success" | "pending"): void {
  status.textContent = message;
  status.dataset.kind = kind;
}

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  if (pending) return;
  const request = { username: username.value, password: password.value, confirmPassword: confirmation.value };
  if (request.username === "" || request.password === "" || request.confirmPassword === "") {
    show("Vui lòng điền đầy đủ cả ba ô.", "error");
    return;
  }
  if (request.username.length > 50 || request.username.includes("\0") || request.password.length > 20) {
    show("Giới hạn tương thích: ID tối đa 50 đơn vị UTF-16, không chứa NUL; mật khẩu tối đa 20 đơn vị UTF-16.", "error");
    return;
  }
  if (request.password !== request.confirmPassword) {
    show("Mật khẩu xác nhận chưa trùng khớp.", "error");
    return;
  }
  pending = true;
  submit.disabled = true;
  show("Đang đăng ký…", "pending");
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), 20000);
  try {
    const response = await fetch("/api/register", {
      method: "POST", headers: { "Content-Type": "application/json" },
      credentials: "omit", mode: "same-origin", cache: "no-store", redirect: "error",
      body: JSON.stringify(request), signal: controller.signal,
    });
    if (response.status === 201) {
      password.value = "";
      confirmation.value = "";
      show("Đăng ký thành công! Quay lại game Flash, nhập đúng ID và mật khẩu rồi tạo nhân vật.", "success");
    } else if (response.status === 409) {
      show("ID này đã tồn tại. Vui lòng chọn ID khác.", "error");
    } else if (response.status === 429) {
      show("Quá nhiều yêu cầu hoặc máy chủ đang bận. Chờ ít nhất 12 giây rồi thử lại; IP VPN dùng chung có chung giới hạn.", "error");
    } else if (response.status === 400 || response.status === 413) {
      show("Dữ liệu không hợp lệ. Kiểm tra giới hạn và mật khẩu xác nhận.", "error");
    } else {
      show("Chưa thể đăng ký. Vui lòng thử lại sau.", "error");
    }
  } catch {
    show("Không xác nhận được kết quả. Kiểm tra kết nối; nếu ID đã tồn tại khi thử lại, hãy đăng nhập bằng thông tin vừa nhập.", "error");
  } finally {
    window.clearTimeout(timeout);
    pending = false;
    submit.disabled = false;
    request.password = "";
    request.confirmPassword = "";
  }
});
