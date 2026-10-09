.. _proto-config-cipher-suites:

Suites de chiffrement TLS
#########################

Les configurations protocolaires des protocoles utilisant TLS version 1.2 ou
inférieure acceptent toutes une option ``cipherSuites``. Celle-ci permet de
restreindre la liste des suites de chiffrement TLS autorisées lors des connexions
avec l'agent concerné.

Si la liste est vide, alors par défaut, seuls les algorithmes "sûrs" (non compromis)
sont acceptés.

Cette option est principalement utile pour l'interopérabilité avec des
partenaires *legacy* (mainframe par exemple) n'acceptant que des suites de
chiffrement spécifiques. De plus, si un algorithme de chiffrement s'avère non
sécurisé à l'avenir, cette option permet également de le désactiver sans avoir
à mettre à jour le programme.

Les suites de chiffrement suivantes sont activées par défaut :

- ``TLS_AES_128_GCM_SHA256``
- ``TLS_AES_256_GCM_SHA384``
- ``TLS_CHACHA20_POLY1305_SHA256``
- ``TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA``
- ``TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA``
- ``TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA``
- ``TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA``
- ``TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256``
- ``TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384``
- ``TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256``
- ``TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384``
- ``TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256``
- ``TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256``

Les suites de chiffrement suivantes sont également acceptées, mais ne sont pas
considérées comme sûres et sont donc désactivées par défaut. Il est donc déconseillé
de les utiliser, sauf si le partenaire distant n'en supporte aucune autre :

- ``TLS_RSA_WITH_RC4_128_SHA``
- ``TLS_RSA_WITH_3DES_EDE_CBC_SHA``
- ``TLS_RSA_WITH_AES_128_CBC_SHA``
- ``TLS_RSA_WITH_AES_256_CBC_SHA``
- ``TLS_RSA_WITH_AES_128_CBC_SHA256``
- ``TLS_RSA_WITH_AES_128_GCM_SHA256``
- ``TLS_RSA_WITH_AES_256_GCM_SHA384``
- ``TLS_ECDHE_ECDSA_WITH_RC4_128_SHA``
- ``TLS_ECDHE_RSA_WITH_RC4_128_SHA``
- ``TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA``
- ``TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256``
- ``TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256``

.. note::
   L'ordre dans lequel les suites de chiffrement sont données n'a pas
   d'importance. Gateway choisit toujours la suite la plus sûre parmi celles
   autorisées des deux côtés de la connexion.

.. note::
   Les suites ``TLS_AES_128_GCM_SHA256``, ``TLS_AES_256_GCM_SHA384`` et
   ``TLS_CHACHA20_POLY1305_SHA256`` sont spécifiques à TLS 1.3. Les suites de
   chiffrement de TLS 1.3 n'étant pas configurables, ces noms sont acceptés mais
   n'ont aucun effet : une connexion en TLS 1.3 utilisera toujours l'une de ces
   3 suites. Par conséquent, une liste ne contenant que ces 3 suites interdira
   de fait toute connexion en TLS 1.2 ou inférieur.

.. note::
   Lors d'un transfert, si le partenaire distant définit une liste de suites de
   chiffrement, celle-ci remplace intégralement celle définie dans la
   configuration du client. Sinon, la liste du client est utilisée.

**Exemple**

.. code-block:: json

   {
     "minTLSVersion": "v1.2",
     "cipherSuites": [
       "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
       "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
     ]
   }
