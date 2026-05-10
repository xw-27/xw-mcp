module.exports = {
    type: 'tool',
    name: 'divide',
    description: '两个数字相除',
    inputSchema: {
        type: 'object',
        properties: {
            dividend: {
                type: 'number',
                description: '被除数'
            },
            divisor: {
                type: 'number',
                description: '除数，不能为0'
            }
        },
        required: ['dividend', 'divisor']
    },
    execute: function(params) {
        if (params.divisor === 0) {
            return { error: '除数不能为0' };
        }
        return { result: params.dividend / params.divisor };
    }
}