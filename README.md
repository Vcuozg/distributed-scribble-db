# Distributed Scribble Database

## 1. Giới thiệu dự án

Dự án này được phát triển dựa trên mã nguồn mở **Scribble**.

Scribble là một cơ sở dữ liệu JSON nhỏ được viết bằng ngôn ngữ Go. Dữ liệu được lưu trực tiếp dưới dạng các file JSON trên ổ đĩa. Dự án gốc hỗ trợ các thao tác cơ bản như ghi dữ liệu, đọc dữ liệu, đọc toàn bộ dữ liệu và xóa dữ liệu.

Repo gốc: https://github.com/sdomino/scribble

## 2. Mục tiêu bài tập lớn

Mục tiêu của bài tập lớn là tìm hiểu, cài đặt và mở rộng Scribble theo hướng hệ phân tán.

Các phần sẽ thực hiện:

- Cài đặt và chạy thử dự án gốc Scribble.
- Kiểm thử các chức năng ban đầu của Scribble.
- Phát triển thêm HTTP API Server.
- Phát triển thêm cơ chế Master-Replica Replication.
- Quản lý mã nguồn bằng GitHub với lịch sử commit rõ ràng.

## 3. Môi trường cài đặt

- Hệ điều hành: Windows
- Ngôn ngữ: Go
- Phiên bản Go: go1.26.3 windows/amd64
- Git: 2.49.0.windows.1
- IDE: Visual Studio Code

## 4. Cài đặt dự án

Clone repository:

```bash
git clone https://github.com/Vcuozg/distributed-scribble-db.git
cd distributed-scribble-db/scribble
```

Cài đặt dependency:

```bash
go mod tidy
```

## 5. Chạy kiểm thử dự án gốc

Chạy lệnh:

```bash
go test
```

Kết quả kiểm thử:

```bash
PASS
ok      github.com/Vcuozg/distributed-scribble-db/scribble      0.487s
```

Kết quả trên cho thấy dự án gốc Scribble đã được cài đặt và kiểm thử thành công.

## 6. Kế hoạch phát triển tính năng mới

### Tính năng 1: HTTP API Server

Mục tiêu là biến Scribble từ thư viện local thành một dịch vụ có thể truy cập qua mạng bằng HTTP API.

Dự kiến API:

- POST /write
- GET /read
- GET /read-all
- DELETE /delete

### Tính năng 2: Master-Replica Replication

Mục tiêu là mô phỏng cơ chế phân tán dữ liệu giữa hai node.

Khi client ghi dữ liệu vào Master Node, dữ liệu sẽ được đồng bộ sang Replica Node.

## 7. Quản lý mã nguồn

Dự án được quản lý bằng GitHub. Mỗi giai đoạn phát triển sẽ được commit riêng để thể hiện quá trình làm bài tập lớn.
