Lister les moniteurs
====================

.. http:get:: /api/snmp/monitors

   Renvoie une liste des moniteurs SNMP.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les moniteurs seront triés *(défaut: name+)*
   :type sort: [name+|name-|address+|address-]

   :statuscode 200: Les moniteurs ont été renvoyés avec succès
   :statuscode 400: Un ou plusieurs des paramètres de la requête sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array monitors: La liste des moniteurs demandés
   :resjsonarr string name: Le nom du moniteur SNMP.
   :resjsonarr string version: La version de SNMP utilisée par le moniteur. Les
      versions acceptées sont "SNMPv2" et "SNMPv3" (SNMPv1 n'est pas supporté).
   :resjsonarr string udpAddress: L'adresse UDP du moniteur à laquelle les
      notifications SNMP doivent être envoyées.
   :resjsonarr bool useInforms: Spécifie le type de notification à envoyer au moniteur.
      Si *faux* (par défaut), Gateway enverra des *traps*. Si *vrai*, Gateway
      enverra des *informs*.
   :resjsonarr string community: [SNMPv2 uniquement] La valeur de communauté
      (ou mot de passe) du moniteur. Par défaut, la valeur "public" est utilisée.
   :resjsonarr string contextName: [SNMPv3 uniquement] Le nom du contexte SNMPv3.
   :resjsonarr string contextEngineID: [SNMPv3 uniquement] L'ID du moteur de contexte SNMPv3.
   :resjsonarr string snmpv3Security: [SNMPv3 uniquement] Spécifie le niveau de
      sécurité SNMPv3 à utiliser avec ce moniteur. Les valeurs acceptées sont :

         - ``noAuthNoPriv``: pas d'authentification ni de confidentialité
         - ``authNoPriv``: authentification, mais pas de confidentialité
         - ``authPriv``: authentification et confidentialité

     Par défaut, l'authentification et la confidentialité sont toutes deux
     désactivées.
   :resjsonarr string authEngineID: [SNMPv3 uniquement] L'ID du moteur d'authentification.
      N'a aucun effet si le moniteur utilise des *informs* (voir l'option *useInforms*
      ci-dessus).
   :resjsonarr string authUsername: [SNMPv3 uniquement] Le nom d'utilisateur. À noter
      que le nom d'utilisateur est requis avec SNMPv3 même si l'authentification
      est désactivée.
   :resjsonarr string authProtocol: [SNMPv3 uniquement] L'algorithme d'authentification
      utilisé. Les valeurs acceptées sont : ``MD5``, ``SHA``, ``SHA-224``, ``SHA-256``,
      ``SHA-384`` et ``SHA-512``.
   :resjsonarr string authPassphrase: [SNMPv3 uniquement] La passphrase d'authentification.
   :resjsonarr string privProtocol: [SNMPv3 uniquement] L'algorithme de confidentialité
      utilisé. Les valeurs acceptées sont : ``DES``, ``AES``, ``AES-192``, ``AES-192C``,
      ``AES-256`` et ``AES-256C``.
   :resjsonarr string privPassphrase: [SNMPv3 uniquement] La passphrase de confidentialité.


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/snmp/monitors HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 482

      {
          "monitors": [{
              "name": "snmpv3-monitor",
              "version": "SNMPv3",
              "udpAddress": "127.0.0.1:162",
              "useInforms": true,
              "snmpv3Security": "authNoPriv",
              "authUsername": "waarp-gw",
              "authProtocol": "SHA",
              "authPassphrase": "sesame"
          }, {
              "name": "snmpv2-monitor",
              "version": "SNMPv2",
              "udpAddress": "192.168.1.1:162",
              "useInforms": false,
              "community": "private"
          }]
      }
