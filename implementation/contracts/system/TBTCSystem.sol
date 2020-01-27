/* solium-disable function-order */
pragma solidity ^0.5.10;

import {IKeepRegistry} from "@keep-network/keep-ecdsa/contracts/api/IKeepRegistry.sol";
import {IECDSAKeepVendor} from "@keep-network/keep-ecdsa/contracts/api/IECDSAKeepVendor.sol";

import {IRelay} from "@summa-tx/relay-sol/contracts/Relay.sol";

import {ITBTCSystem} from "../interfaces/ITBTCSystem.sol";
import {IBTCETHPriceFeed} from "../interfaces/IBTCETHPriceFeed.sol";
import {DepositLog} from "../DepositLog.sol";

import "openzeppelin-solidity/contracts/ownership/Ownable.sol";

contract TBTCSystem is Ownable, ITBTCSystem, DepositLog {

    bool _initialized = false;

    address public keepRegistry;
    address public priceFeed;
    address public relay;

    // Parameters governed by the TBTCSystem owner
    bool private allowNewDeposits = true;
    uint256 private signerFeeDivisor = 200; // 1/200 == 50bps == 0.5% == 0.005
    uint128 private undercollateralizedThresholdPercent = 140;  // percent
    uint128 private severelyUndercollateralizedThresholdPercent = 120; // percent

    constructor(address _priceFeed, address _relay) public {
        priceFeed = _priceFeed;
        relay = _relay;
    }

    function initialize(
        address _keepRegistry
    ) external onlyOwner {
        require(!_initialized, "already initialized");

        keepRegistry = _keepRegistry;
        _initialized = true;
    }

    /// @notice Enables/disables new deposits from being created.
    /// @param _allowNewDeposits Whether to allow new deposits.
    function setAllowNewDeposits(bool _allowNewDeposits)
        external onlyOwner
    {
        allowNewDeposits = _allowNewDeposits;
    }

    /// @notice Gets whether new deposits are allowed.
    function getAllowNewDeposits() external view returns (bool) { return allowNewDeposits; }

    /// @notice Set the system signer fee divisor.
    /// @param _signerFeeDivisor The signer fee divisor.
    function setSignerFeeDivisor(uint256 _signerFeeDivisor)
        external onlyOwner
    {
        require(_signerFeeDivisor > 1, "Signer fee must be lower than 100%");
        signerFeeDivisor = _signerFeeDivisor;
    }

    /// @notice Gets the system signer fee divisor.
    /// @return The signer fee divisor.
    function getSignerFeeDivisor() external view returns (uint256) { return signerFeeDivisor; }

    /// @notice Set the system collateralization levels
    /// @param _undercollateralizedThresholdPercent first undercollateralization trigger
    /// @param _severelyUndercollateralizedThresholdPercent second undercollateralization trigger
    function setCollateralizationThresholds(
        uint128 _undercollateralizedThresholdPercent,
        uint128 _severelyUndercollateralizedThresholdPercent
    ) external onlyOwner {
        require(
            _undercollateralizedThresholdPercent > _severelyUndercollateralizedThresholdPercent,
            "Severe undercollateralized threshold must be > undercollateralized threshold"
        );
        undercollateralizedThresholdPercent = _undercollateralizedThresholdPercent;
        severelyUndercollateralizedThresholdPercent = _severelyUndercollateralizedThresholdPercent;
    }

    /// @notice Get the system undercollateralization level for new deposits
    function getUndercollateralizedThresholdPercent() external view returns (uint128) {
        return undercollateralizedThresholdPercent;
    }

    /// @notice Get the system severe undercollateralization level for new deposits
    function getSeverelyUndercollateralizedThresholdPercent() external view returns (uint128) {
        return severelyUndercollateralizedThresholdPercent;
    }

    // Price Feed
    function fetchBitcoinPrice() external view returns (uint256) {
        return IBTCETHPriceFeed(priceFeed).getPrice();
    }

    // Difficulty Oracle
    // TODO: This is a workaround. It will be replaced by tbtc-difficulty-oracle.
    function fetchRelayCurrentDifficulty() external view returns (uint256) {
        return IRelay(relay).getCurrentEpochDifficulty();
    }

    function fetchRelayPreviousDifficulty() external view returns (uint256) {
        return IRelay(relay).getPrevEpochDifficulty();
    }

    /// @notice Request a new keep opening.
    /// @param _m Minimum number of honest keep members required to sign.
    /// @param _n Number of members in the keep.
    /// @return Address of a new keep.
    function requestNewKeep(uint256 _m, uint256 _n)
        external
        payable
        returns (address _keepAddress)
    {
        address keepVendorAddress = IKeepRegistry(keepRegistry)
            .getVendor("ECDSAKeep");

        _keepAddress = IECDSAKeepVendor(keepVendorAddress)
            .openKeep(_n,_m, msg.sender);
    }
}
