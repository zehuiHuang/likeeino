###  环境依赖

```shell
#milvus 容器启动
wget https://github.com/milvus-io/milvus/releases/download/v2.6.7/milvus-standalone-docker-compose.yml -O docker-compose.yml
sudo docker compose up -d
```



### 注意,如果您想要执行案例,需要提供一个.env的文件,里面主要配置一些大模型或trace相关的的配置信息

eino官网地址:https://www.cloudwego.io/zh/docs/eino/

# agent: reactAgent、workflowAgents、multiAgent

# reactAgent:
模型和工具结合,是一个智能决策的大脑,常用场景知识问答、IT运维(可根据线索一步步定位问题)

# workflowAgents: 
允许开发者以预设的流程来组织和执行多个子 Agent,定制化流水线,按照顺序严格执行,常用场景比如CICD、数据迁移等
三种类型:
1、SequentialAgent：按顺序依次执行子 Agent
案例:

2、LoopAgent：循环执行子 Agent 序列

3、ParallelAgent：并发执行多个子 Agent

# multiAgent:
1、集中式协调(Supervisor),中心化设计,主agent管理一批子agent,主agent可以根据情况来进行动态调整和任务分配给子agent:
Supervisor 案例:adk/multiagent/supervisor

2、结构化问题解决(Plan-Execute),种基于「规划-执行-反思」范式的多智能体协作框架,旨在解决复杂任务的分步拆解、执行与动态调整问题,主要包括:
规划器、执行器和重规划器:
Plan-Execute 案例:adk/multiagent/plan-execute-replan




^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^^_^各组件定义和使用

---------------------------------------------------------------Graph 编排
一、Graph 编排
1、使用时
将各个组件先添加到graph,然后通过AddEdge将各个组件串联起来,首尾分别是:compose.START和compose.END,其他组件(名称)在中间,
并且按照头尾相连.
各组件包括:AddChatTemplateNode、AddChatModelNode、AddToolsNode、
AddLambdaNode、AddRetrieverNode、AddEdge、AddBranch、AddDocumentTransformerNode、AddEmbeddingNode、AddIndexerNode等

2、全局状态

---------------------------------------------------------------Graph 编排





---------------------------------------------------------------人机协同
//代码演示路径adk/human-in-the-loop/
二、人机协同
# 背景说明:
在程序执行的过程中,需要人的参与(比如确认信息是否正确、内容是否需要优化),
通过外部的参数接入来影响程序后续的执行(中断、执行、调整重新执行、循环执行等)
# 场景案例
1、确认信息,根据信息人为的中断或继续
2、修改信息,参考工具返回的参数信息,对参数信息进行调整后在执行
3、给agent返回的数据人为的提出复合自身要求的描述,agent会以此为条件执行并给出结果,并无限循环,直至人为要求的进行最终答案的输出
4、通过多轮对话,识别用户对话,制定出更符合预期的答案(常适用于咨询、问题收集等场景)

---------------------------------------------------------------人机协同