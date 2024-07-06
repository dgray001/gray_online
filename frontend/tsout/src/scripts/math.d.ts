export declare function modulus(n: number, m: number): number;
export declare function atangent(y: number, x: number): number;
export declare function clampNumber(n: number, min: number, max: number, default_min?: boolean): number;
export declare interface BoundedNumber {
    value_min: number;
    value_max: number;
    value: number;
}
export declare function validateBoundedNumber(n: BoundedNumber): BoundedNumber;
export declare function setBoundedNumber(n: BoundedNumber, v: number): number;
