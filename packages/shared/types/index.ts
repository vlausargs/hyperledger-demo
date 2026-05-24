export interface Product {
  docType: string;
  id: string;
  sku: string;
  name: string;
  description: string;
  batchId: string;
  manufacturerId: string;
  manufacturerName: string;
  manufacturedAt: string;
  expiryDate: string;
  status: 'ACTIVE' | 'SHIPPED' | 'DELIVERED' | 'RECALLED' | 'SCRAPPED' | 'SOLD';
  currentOwnerMSP: string;
  currentOwner: string;
  currentShipmentId: string;
  recallId: string;
  metadata: Record<string, string>;
  createdAt: string;
  updatedAt: string;
}

export interface Shipment {
  docType: string;
  id: string;
  name: string;
  description: string;
  productIds: string[];
  senderMsp: string;
  senderName: string;
  receiverMsp: string;
  receiverName: string;
  status: 'DRAFT' | 'IN_TRANSIT' | 'DELIVERED' | 'RECALLED' | 'CANCELLED';
  origin: string;
  destination: string;
  departureAt: string;
  arrivalAt: string;
  recallId: string;
  createdAt: string;
  updatedAt: string;
}

export interface CustodyRecord {
  docType: string;
  id: string;
  shipmentId: string;
  sequence: number;
  fromMsp: string;
  fromName: string;
  toMsp: string;
  toName: string;
  status: 'PENDING' | 'COMPLETED' | 'REJECTED';
  transferredAt: string;
  conditions: string;
  senderSignature: string;
  receiverSignature: string;
  txId: string;
  createdAt: string;
  updatedAt: string;
}

export interface SupplyChainEvent {
  docType: string;
  id: string;
  targetId: string;
  targetType: string;
  eventType: string;
  description: string;
  location: string;
  recordedBy: string;
  data: Record<string, string>;
  occurredAt: string;
  txId: string;
  createdAt: string;
}

export interface RecallNotice {
  docType: string;
  id: string;
  scope: 'PRODUCT' | 'SHIPMENT' | 'BATCH';
  targetIds: string[];
  reason: string;
  issuedBy: string;
  issuedByName: string;
  severity: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
  instructionsUrl: string;
  affectedCount: number;
  createdAt: string;
}

export interface SaleTransaction {
  docType: string;
  id: string;
  items: SaleItem[];
  customerId: string;
  cashierId: string;
  cashierName: string;
  subTotal: number;
  taxAmount: number;
  totalAmount: number;
  currency: string;
  retailerMsp: string;
  notes: string;
  txId: string;
  createdAt: string;
}

export interface SaleItem {
  productId: string;
  sku: string;
  productName: string;
  unitPrice: number;
}

export interface InventoryItem {
  sku: string;
  name: string;
  count: number;
  productIds: string[];
}

export interface PagedResult<T> {
  bookmark: string;
  count: number;
}

export interface PagedProductResult extends PagedResult<Product> {
  products: Product[];
}

export interface PagedShipmentResult extends PagedResult<Shipment> {
  shipments: Shipment[];
}

export interface PagedSaleResult extends PagedResult<SaleTransaction> {
  sales: SaleTransaction[];
}

export interface HistoryQueryResult {
  txId: string;
  timestamp: string;
  isDelete: boolean;
  value: unknown;
}

export interface ProvenanceResult {
  product: Product;
  history: HistoryQueryResult[];
  custody: CustodyRecord[];
  events: SupplyChainEvent[];
}

export type ProductStatus = Product['status'];
export type ShipmentStatus = Shipment['status'];
export type CustodyStatus = CustodyRecord['status'];
export type RecallScope = RecallNotice['scope'];
export type RecallSeverity = RecallNotice['severity'];
