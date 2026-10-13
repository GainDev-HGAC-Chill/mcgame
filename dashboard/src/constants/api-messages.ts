export const API_MESSAGES = {
    common: {
        apiError: "Lỗi API",
        unexpectedError: "Đã xảy ra lỗi không mong muốn",
        invalidRequestPayload: "Dữ liệu yêu cầu không hợp lệ",
        invalidRequestPayloadLowercase: "Dữ liệu yêu cầu không hợp lệ",
        invalidRequest: "Yêu cầu không hợp lệ",
        unknownError: "Lỗi không xác định",
        unknownErrorLowercase: "Lỗi không xác định",
        missingDatabaseURL:
            "Kết nối cơ sở dữ liệu thất bại. Kiểm tra cấu hình Supabase.",
    },
    admin: {
        loginRequired: "Cần đăng nhập quản trị",
        adminSecretRequired: "Cần khóa bí mật quản trị",
        adminServerEnvRequired:
            "Thiết lập GAME_SERVER_HTTP_URL hoặc ADMIN_SERVER_API_URL trong môi trường bảng quản trị",
        failedToReachAdminServer: "Không kết nối được máy chủ quản trị",
    },
    creatures: {
        monsterApprFileNotFound: "Không tìm thấy monster_appr.json",
        monsterApprParseFailed: "Không đọc được monster_appr.json",
        imageDirectoryNotFound: "Không tìm thấy thư mục hình sinh vật",
        imageDirectoryScanFailed: "Không quét được thư mục hình ảnh",
    },
    boxItems: {
        loadFailed: "Không tải được hộp vật phẩm",
        awardsLoadFailed: "Không tải được phần thưởng vật phẩm",
        awardSaveFailed: "Không lưu được phần thưởng vật phẩm",
        awardDeleteFailed: "Không xóa được phần thưởng vật phẩm",
        awardOptionsLoadFailed: "Không tải được lựa chọn phần thưởng",
        invalidItemId: "ID vật phẩm phải là số nguyên dương",
        invalidAwardId: "ID phần thưởng phải là số nguyên dương",
        invalidAwardData: "Dữ liệu phần thưởng không hợp lệ",
        invalidAwardType: "Loại phần thưởng chưa được hỗ trợ",
    },
    dailySigninRewards: {
        loadFailed: "Không tải được phần thưởng điểm danh",
        saveFailed: "Không lưu được phần thưởng điểm danh",
        deleteFailed: "Không xóa được phần thưởng điểm danh",
        invalidPayload: "Dữ liệu phần thưởng không hợp lệ",
        invalidInc: "Hệ số inc phải là 1, 2 hoặc 4",
        invalidItemId: "ID vật phẩm phải là số nguyên dương",
        invalidQuantity: "Số lượng phải là số nguyên dương",
        invalidRewardId: "ID phần thưởng phải là số nguyên dương",
    },
    items: {
        dressPanelOptionsLoadFailed: "Không tải được lựa chọn thời trang",
        equipSuitDefaultsRequestInvalid:
            "ID vật phẩm và ID bộ trang bị phải là số nguyên dương.",
        equipSuitDefaultsLoadFailed: "Không tải được mặc định bộ trang bị",
        gemOptionsLoadFailed: "Không tải được lựa chọn đá khảm",
    },
    map: {
        chunksLoadFailed: "Không tải được các phần bản đồ",
        tilesLoadFailed: "Không tải được dữ liệu bản đồ",
    },
    giftCodes: {
        giftCodeLengthInvalid: "Mã quà tặng phải có 4–64 ký tự",
        giftCodeCharactersInvalid: "Mã quà tặng chứa ký tự không hợp lệ",
        campaignKeyTooLong: "Khóa chiến dịch quá dài",
        campaignKeyCharactersInvalid:
            "Khóa chiến dịch chứa ký tự không hợp lệ",
        timestampInvalid: "Thời gian không hợp lệ",
        rewardsRequired: "Cần ít nhất một phần thưởng",
        rewardTypeRequired: "Vui lòng chọn loại phần thưởng",
        itemIDRequired: "Vui lòng nhập ID vật phẩm",
        codesOrGeneratorRequired:
            "Cần ít nhất một mã hoặc cấu hình tự tạo mã",
        codesAndGeneratorConflict: "Chọn nhập mã hoặc tự tạo mã, không dùng cả hai",
        campaignIDRequired: "Vui lòng nhập ID chiến dịch",
        activeRequired: "Vui lòng chọn trạng thái kích hoạt",
        campaignNotFound: "Không tìm thấy chiến dịch",
        updatedCampaignNotFound: "Không tìm thấy chiến dịch vừa cập nhật",
        createdCampaignNotFound: "Không tìm thấy chiến dịch vừa tạo",
        campaignNameRequired: "Vui lòng nhập tên chiến dịch",
        endsAtBeforeStartsAt: "Thời gian kết thúc phải sau thời gian bắt đầu",
        usageLimitsInvalid: "Giới hạn lượt dùng không được âm",
        codeMaxUsesInvalid: "Lượt dùng tối đa của mã không được âm",
        redeemedCampaignCodesImmutable:
            "Không thể sửa mã của chiến dịch đã có lượt nhận",
        redeemedCampaignDeleteForbidden: "Không thể xóa chiến dịch đã có lượt nhận",
        campaignExists: "Khóa chiến dịch đã tồn tại",
        codeExists: "Mã quà tặng đã tồn tại",
    },
    sendPlayerItem: {
        invalidOptionsObject: "Tùy chọn phải là đối tượng",
        invalidTemplateTableId: "ID bảng mẫu phải là 19 hoặc 29",
        invalidPropLinesArray: "Danh sách dòng thuộc tính phải là mảng",
        invalidPropLineItem: "Mỗi dòng thuộc tính phải là đối tượng",
        invalidPropLineSlot:
            "Ô thuộc tính phải là main1, main2, prop1, prop2 hoặc active",
        invalidPropLineType: "Loại thuộc tính phải là số nguyên dương",
        invalidPropLineValue: "Giá trị thuộc tính phải từ 0 trở lên",
        invalidGemsArray: "Danh sách đá khảm phải là mảng",
        invalidGemValue: "Mọi ID đá khảm phải là số nguyên dương",
        invalidRawPropertiesObject: "Thuộc tính thô phải là đối tượng",
        invalidColor: "Mã màu phải từ 0 trở lên",
        invalidStrengthenLevel: "Cấp cường hóa phải từ 0 trở lên",
        invalidEndureLeft: "Độ bền còn lại phải từ 0 trở lên",
        invalidEndureMax: "Độ bền tối đa phải từ 0 trở lên",
        invalidElement: "Nguyên tố phải từ 0 trở lên",
        invalidPreNameType: "Loại tiền tố phải từ 0 trở lên",
        invalidHoleNum: "Số ô khảm phải từ 0 đến 10",
        invalidBindMainPropNum:
            "Hai giá trị thưởng thuộc tính khóa phải từ 0 trở lên",
        invalidPlayerID: "ID người chơi phải là số nguyên dương",
        invalidItemID: "ID vật phẩm phải là số nguyên dương",
        invalidCount: "Số lượng phải là số nguyên dương",
        countTooLarge: "Số lượng quá lớn",
        itemSavedToDatabase:
            "Vật phẩm đã lưu vào cơ sở dữ liệu. Nhân vật trực tuyến có thể cần đăng nhập lại hoặc làm mới túi đồ.",
    },
    client: {
        giftCodeCampaignsLoadFailed: "Không tải được chiến dịch mã quà tặng",
        giftCodeCampaignCreateFailed: "Không tạo được chiến dịch",
        giftCodeCampaignUpdateFailed: "Không cập nhật được chiến dịch",
        giftCodeCampaignStatusUpdateFailed: "Không cập nhật được trạng thái chiến dịch",
        giftCodeCampaignDeleteFailed: "Không xóa được chiến dịch",
    },
} as const

export const API_MESSAGE_BUILDERS = {
    giftCodes: {
        rewardAmountMustBePositive: (rewardType: string) =>
            `Số lượng phần thưởng phải dương đối với ${rewardType}`,
        unsupportedRewardType: (rewardType: string) =>
            `Loại phần thưởng chưa được hỗ trợ: ${rewardType}`,
        duplicateCode: (code: string) => `Mã bị trùng: ${code}`,
        generateCountOutOfRange: (maxCount: number) =>
            `Số mã tự tạo phải từ 1 đến ${maxCount}`,
        missingItemTemplate: (itemID: number) =>
            `Không tìm thấy mẫu vật phẩm phần thưởng: ${itemID}`,
        resolveServerKeyFailed: (serverKey: string) =>
            `Không tra cứu được khóa máy chủ ${serverKey}`,
    },
    sendPlayerItem: {
        playerNotFound: (playerID: number) => `Không tìm thấy nhân vật ${playerID}`,
        itemTemplateNotFound: (itemID: number, templateTableId?: number) =>
            templateTableId
                ? `Không tìm thấy mẫu vật phẩm ${itemID} trong bảng ${templateTableId}`
                : `Không tìm thấy mẫu vật phẩm ${itemID}`,
        inventoryFull: (requiredSlots: number, availableSlots: number) =>
            `Túi đồ đã đầy. Cần ${requiredSlots} ô, còn ${availableSlots} ô.`,
    },
} as const
