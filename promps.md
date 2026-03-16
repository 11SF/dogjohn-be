GET /api/v1/payment/details
  Response:
    promptPayId: string           # เลข PromptPay แสดงใต้ QR
    prices:
      - id: string
        price: number
        description: string       # "นิดหน่อย", "พอดี", "เยอะ"

GET /api/v1/feeder/availability   # เปลี่ยนจาก /order/status
  Response:
    available: boolean            # true = เปิดรับ, false = ปิด (disable UI)
    reason: string | null         # เหตุผลที่ปิด เช่น "เครื่องขัดข้อง", "อาหารหมด"

POST /api/v1/order/submit
  Request (multipart/form-data):
    priceId: string
    customerName: string
    slipImage: file
  Response:
    orderId: string               # ใช้ poll status ต่อ

GET /api/v1/order/:orderId
  Response:
    orderId: string
    customerName: string
    priceDetails:
      price: string               # "10.00"
      description: string
    orderStatus: PENDING          # กำลังตรวจสลิป
                | PROCESSING      # ตรวจผ่าน กำลังให้อาหาร
                | COMPLETED       # เสร็จแล้ว
                | FAILED          # สลิปไม่ผ่าน / เครื่องพัง
    failureReason: string | null  # กรณี FAILED บอกสาเหตุ
    orderedAt: number             # unix timestamp

GET /api/v1/order/summary
  Response:
    today: number                 # จำนวนครั้งที่ให้วันนี้
    dogCount: number              # จำนวนน้องหมา
    totalAmount: number           # เปลี่ยนจาก totalPrices → ชื่อชัดกว่า

GET /api/v1/order/history
  Query params:
    limit: number (default 20)
    offset: number (default 0)   
  Response:
    items:
      - orderId: string
        customerName: string
        priceDetails:
          price: string
          description: string
        orderStatus: string
        orderedAt: number
    total: number                 # จำนวนทั้งหมด ใช้ทำ pagination
```

---

**Flow หลักของ UI:**
```
1. GET /feeder/availability     → ถ้า false = disable ปุ่ม + แสดง reason
2. GET /payment/details         → โหลด QR + ราคา
3. POST /order/submit           → ได้ orderId กลับมา
4. Polling GET /order/:orderId  → ทุก 2-3 วิ จนได้ COMPLETED หรือ FAILED
5. GET /order/summary           → refresh stats
6. GET /order/history           → refresh log ล่าสุด