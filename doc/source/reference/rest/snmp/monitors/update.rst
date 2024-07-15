Modifier un moniteur
====================

.. http:patch:: /api/snmp/monitors/{string:monitor}

   Modifie le moniteur SNMP demandé avec les informations renseignées en format
   JSON dans le corps de la requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du moniteur SNMP.
   :reqjson string version: La version de SNMP utilisée par le moniteur. Les
      versions acceptées sont "SNMPv2" et "SNMPv3" (SNMPv1 n'est pas supporté).
   :reqjson string udpAddress: L'adresse UDP du moniteur à laquelle les
      notifications SNMP doivent être envoyées.
   :reqjson bool useInforms: Spécifie le type de notification à envoyer au moniteur.
      Si *faux* (par défaut), Gateway enverra des *traps*. Si *vrai*, Gateway
      enverra des *informs*.
   :reqjson string community: [SNMPv2 uniquement] La valeur de communauté
      (ou mot de passe) du moniteur. Par défaut, la valeur "public" est utilisée.
   :reqjson string contextName: [SNMPv3 uniquement] Le nom du contexte SNMPv3.
   :reqjson string contextEngineID: [SNMPv3 uniquement] L'ID du moteur de contexte SNMPv3.
   :reqjson string snmpv3Security: [SNMPv3 uniquement] Spécifie le niveau de
      sécurité SNMPv3 à utiliser avec ce moniteur. Les valeurs acceptées sont :

         - ``noAuthNoPriv``: pas d'authentification ni de confidentialité
         - ``authNoPriv``: authentification, mais pas de confidentialité
         - ``authPriv``: authentification et confidentialité

     Par défaut, l'authentification et la confidentialité sont toutes deux
     désactivées.
   :reqjson string authEngineID: [SNMPv3 uniquement] L'ID du moteur d'authentification.
      N'a aucun effet si le moniteur utilise des *informs* (voir l'option *useInforms*
      ci-dessus).
   :reqjson string authUsername: [SNMPv3 uniquement] Le nom d'utilisateur. À noter
      que le nom d'utilisateur est requis avec SNMPv3 même si l'authentification
      est désactivée.
   :reqjson string authProtocol: [SNMPv3 uniquement] L'algorithme d'authentification
      utilisé. Les valeurs acceptées sont : ``MD5``, ``SHA``, ``SHA-224``, ``SHA-256``,
      ``SHA-384`` et ``SHA-512``.
   :reqjson string authPassphrase: [SNMPv3 uniquement] La passphrase d'authentification.
   :reqjson string privProtocol: [SNMPv3 uniquement] L'algorithme de confidentialité
      utilisé. Les valeurs acceptées sont : ``DES``, ``AES``, ``AES-192``, ``AES-192C``,
      ``AES-256`` et ``AES-256C``.
   :reqjson string privPassphrase: [SNMPv3 uniquement] La passphrase de confidentialité.

   :statuscode 201: Le moniteur a été mis à jour avec succès
   :statuscode 400: Un ou plusieurs des paramètres du moniteur sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le moniteur demandé n'existe pas

   :resheader Location: Le chemin d'accès au nouveau moniteur créé.


   **Exemple de requête**

   .. code-block:: http

      POST https://my_waarp_gateway.net/api/snmp/monitors HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
      Content-Type: application/json
      Content-Length: 174

      {
        "name": "snmp-monitor",
        "version": "SNMPv3",
        "udpAddress": "127.0.0.1:162",
        "useInforms": true,
        "snmpv3Security": "authNoPriv",
        "authUsername": "waarp-gw",
        "authProtocol": "SHA",
        "authPassphrase": "sesame"
      }

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 201 CREATED
      Location: https://my_waarp_gateway.net/api/snmp/monitors/snmp-monitor
