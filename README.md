# Distributed Scribble Database

## 1. Giới thiệu dự án

Distributed Scribble Database là dự án được phát triển dựa trên mã nguồn mở Scribble viết bằng ngôn ngữ Go.

Scribble là một cơ sở dữ liệu JSON nhỏ gọn, lưu trữ dữ liệu trực tiếp dưới dạng các file JSON trên ổ đĩa. Dự án hỗ trợ các thao tác cơ bản như ghi dữ liệu, đọc dữ liệu và xóa dữ liệu mà không cần sử dụng các hệ quản trị cơ sở dữ liệu như MySQL hay PostgreSQL.

Mã nguồn gốc:

https://github.com/sdomino/scribble

Mục tiêu của bài tập lớn là tìm hiểu mã nguồn Scribble, triển khai thành công hệ thống và mở rộng thêm các tính năng liên quan đến hệ phân tán.

---

## 2. Mục tiêu bài tập lớn

Các mục tiêu chính của dự án:

- Tìm hiểu kiến trúc và mã nguồn Scribble.
- Cài đặt và kiểm thử dự án gốc.
- Xây dựng lớp REST API để truy cập dữ liệu qua mạng.
- Xây dựng cơ chế Master – Replica Replication.
- Mô phỏng hoạt động của một hệ thống cơ sở dữ liệu phân tán.
- Quản lý mã nguồn bằng GitHub với lịch sử commit rõ ràng.

---

## 3. Môi trường phát triển

| Thành phần         | Phiên bản              |
| ------------------ | ---------------------- |
| Hệ điều hành       | Windows                |
| Ngôn ngữ lập trình | Go                     |
| Go Version         | go1.26.3 windows/amd64 |
| Git Version        | 2.49.0.windows.1       |
| IDE                | Visual Studio Code     |

---

## 4. Cấu trúc dự án

```text
distributed-scribble-db
│
├── README.md
├── scribble
│   ├── scribble.go
│   ├── scribble_test.go
│   └── ...
│
└── server
    ├── main.go
    ├── data
    └── ...
```

---

## 5. Cài đặt dự án

Clone mã nguồn:

```bash
git clone https://github.com/Vcuozg/distributed-scribble-db.git
```

Di chuyển vào thư mục Scribble:

```bash
cd distributed-scribble-db/scribble
```

Cài đặt dependency:

```bash
go mod tidy
```

---

## 6. Kiểm thử dự án gốc

Chạy lệnh:

```bash
go test
```

Kết quả:

```bash
PASS
ok github.com/Vcuozg/distributed-scribble-db/scribble
```

Kết quả trên cho thấy dự án Scribble gốc đã được cài đặt và kiểm thử thành công.

---

## 7. Kiến trúc hệ thống

### Mô hình tổng thể

```text
                Client
                   |
                   v
        +-------------------+
        |   Master Node     |
        | localhost:8080    |
        +-------------------+
                   |
                   | Replication
                   v
        +-------------------+
        |   Replica Node    |
        | localhost:8081    |
        +-------------------+
```

### Mô tả hoạt động

- Client gửi yêu cầu tới Master Node.
- Master Node lưu dữ liệu vào cơ sở dữ liệu Scribble.
- Sau khi ghi thành công, Master tự động gửi dữ liệu tới Replica Node.
- Replica Node nhận dữ liệu và lưu vào cơ sở dữ liệu riêng.
- Hai node duy trì dữ liệu đồng bộ với nhau.

Mô hình này mô phỏng cơ chế sao chép dữ liệu trong hệ thống phân tán.

---

## 8. Tính năng mới 1: REST API Server

Trong dự án gốc, Scribble chỉ hoạt động như một thư viện Go.

Dự án đã được mở rộng bằng cách xây dựng HTTP REST API để cho phép các ứng dụng khác truy cập dữ liệu thông qua mạng.

### Các API đã triển khai

#### Ghi dữ liệu

```http
POST /write
```

Ví dụ:

```json
{
  "collection": "users",
  "resource": "user1",
  "data": {
    "name": "Cuong",
    "age": 21
  }
}
```

#### Đọc dữ liệu

```http
GET /read?collection=users&resource=user1
```

#### Xóa dữ liệu

```http
DELETE /delete?collection=users&resource=user1
```

### Lợi ích

- Truy cập dữ liệu từ xa thông qua HTTP.
- Dễ dàng tích hợp với các ứng dụng khác.
- Tạo nền tảng cho việc xây dựng hệ thống phân tán.

---

## 9. Tính năng mới 2: Master – Replica Replication

### Mục tiêu

Mô phỏng cơ chế đồng bộ dữ liệu giữa nhiều node trong hệ thống phân tán.

### Luồng hoạt động

```text
Client
   |
POST /write
   |
   v
Master Node
   |
Lưu dữ liệu
   |
Replication
   |
   v
Replica Node
   |
Lưu dữ liệu bản sao
```

### API nội bộ

```http
POST /replicate
```

API này được sử dụng nội bộ để Master gửi dữ liệu sang Replica.

### Ưu điểm

- Tăng khả năng dự phòng dữ liệu.
- Giảm nguy cơ mất dữ liệu.
- Là nền tảng cho các hệ thống phân tán lớn hơn.

---

## 10. Kết quả thực nghiệm

### Kiểm thử chức năng ghi dữ liệu

API:

```http
POST /write
```

Kết quả:

```json
{
  "message": "Data written successfully"
}
```

Trạng thái:

Thành công.

---

### Kiểm thử chức năng đọc dữ liệu

API:

```http
GET /read
```

Kết quả:

```json
{
  "data": {
    "name": "Cuong",
    "age": 21
  }
}
```

Trạng thái:

Thành công.

---

### Kiểm thử chức năng xóa dữ liệu

API:

```http
DELETE /delete
```

Kết quả:

```json
{
  "message": "Data deleted successfully"
}
```

Trạng thái:

Thành công.

---

### Kiểm thử Replication

Bước 1:

Ghi dữ liệu vào Master Node tại:

```text
localhost:8080
```

Bước 2:

Master Node tự động gửi dữ liệu sang Replica Node.

Bước 3:

Đọc dữ liệu tại:

```text
localhost:8081
```

Kết quả:

Dữ liệu xuất hiện trên Replica Node.

Kết luận:

Cơ chế Master – Replica Replication hoạt động thành công.

---

## 11. Quản lý mã nguồn GitHub

Repository:

https://github.com/Vcuozg/distributed-scribble-db

Dự án được phát triển theo từng giai đoạn và được quản lý bằng GitHub.

Các commit chính:

- Import original Scribble source code
- Configure Go module
- Verify original Scribble tests
- Create HTTP API server
- Add write record endpoint
- Add read record endpoint
- Add delete record endpoint
- Add server configuration
- Add replica internal write endpoint
- Implement master replica data replication

Lịch sử commit thể hiện đầy đủ quá trình phát triển dự án.

---

## 12. Kết luận

Dự án đã nghiên cứu và triển khai thành công cơ sở dữ liệu Scribble.

Hai tính năng mới đã được phát triển:

1. REST API Server.
2. Master – Replica Replication.

Kết quả đạt được cho thấy hệ thống có thể mô phỏng các thành phần cơ bản của một hệ thống cơ sở dữ liệu phân tán, bao gồm giao tiếp qua mạng, lưu trữ dữ liệu và đồng bộ dữ liệu giữa các node.
