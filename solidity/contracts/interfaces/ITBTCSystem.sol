pragma solidity ^0.5.10;

/**
 * @title Keep interface
 */

interface ITBTCSystem {

    // expected behavior:
    // return the price of 1 sat in wei
    // these are the native units of the deposit contract
    function fetchBitcoinPrice() external view returns (uint256);

    // passthrough requests for the oracle
    function fetchRelayCurrentDifficulty() external view returns (uint256);
    function fetchRelayPreviousDifficulty() external view returns (uint256);

    function getInitialCollateralizedPercent() external view returns (uint128);
    function getAllowNewDeposits() external view returns (bool);
    function isAllowedLotSize(uint256 _lotSizeSatoshis) external view returns (bool);
    function requestNewKeep(uint256 _m, uint256 _n, uint256 _bond) external payable returns (address);
    function getSignerFeeDivisor() external view returns (uint256);
    function getUndercollateralizedThresholdPercent() external view returns (uint128);
    function getSeverelyUndercollateralizedThresholdPercent() external view returns (uint128);
}