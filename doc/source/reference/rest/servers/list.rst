Lister les serveurs
======================

.. http:get:: /api/servers

   .. deprecated:: 0.5.0
      
      * Les propriétés ``indir`` et `outDir`` de la réponse ont été remplacées par
        les propriétés ``sendDir`` et ``receiveDir``.
      * La propriété ``root``de la réponse a été remplacée par la propriété
        ``rootDir``.
      * La propriété ``workDir` de la réponse a été remplacée par la propriété
        ``tmpReceiveDir``.

   Renvoie une liste des serveurs remplissant les critères donnés en paramètres
   de requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les serveurs seront triés.
      Valeurs possibles : ``name+``, ``name-``, ``protocol+``, ``protocol-``.
      *(défaut: name+)*
   :type sort: string
   :param protocol: Filtre uniquement les serveurs utilisant ce protocole.
      Peut être renseigné plusieurs fois pour filtrer plusieurs protocoles.
   :type protocol: string

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array servers: La liste des serveurs demandés
   :resjsonarr string name: Le nom du serveur
   :resjsonarr string protocol: Le protocole utilisé par le serveur
   :resjsonarr string address: L'adresse du serveur (en format [adresse:port])
   :resjsonarr bool enabled: Indique si le serveur est activé ou non au démarrage
      de Gateway.
   :resjsonarr string rootDir: Chemin du dossier racine du serveur. Peut être
      relatif (à la racine de la *gateway*) ou absolu.
   :resjsonarr string receiveDir: Le dossier de réception du serveur. Peut
      être relatif (à la racine du serveur) ou absolu.
   :resjsonarr string sendDir: Le dossier d'envoi du serveur. Peut être
      relatif (à la racine du serveur) ou absolu.
   :resjsonarr string tmpReceiveDir: Le dossier temporaire du serveur. Peut
      être relatif (à la racine du serveur) ou absolu.
   :resjsonarr array authMethods: La liste des valeurs utilisées par le serveur
      pour s'authentifier auprès des clients externes qui s'y connectent.
   :resjsonarr object protoConfig: La configuration du serveur encodé sous forme
      d'un objet JSON. Cet objet dépend du protocole.
   :resjsonarr object authorizedRules: Les règles que le serveur est autorisé à
      utiliser pour les transferts.

      * ``sending`` (*array* of *string*) - Les règles d'envoi.
      * ``reception`` (*array* of *string*) - Les règles de réception.

   :resjsonarr string root: *Déprécié*. La racine du serveur. Peut être relatif
      (à la racine de la *gateway*) ou absolu .
   :resjsonarr string inDir: *Déprécié*. Le dossier de réception du serveur.
      Peut être relatif (à la racine du serveur) ou absolu. 
   :resjsonarr string outDir: *Déprécié*. Le dossier d'envoi du serveur. Peut
      être relatif (à la racine du serveur) ou absolu. 
   :resjsonarr string workDir: *Déprécié*. Le dossier temporaire du serveur.
      Peut être relatif (à la racine du serveur) ou absolu. 

   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/servers?limit=10&sort=name-&protocol=sftp HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 619

      {
        "servers": [{
          "name": "sftp_server_2",
          "protocol": "sftp",
          "address": "localhost:2023",
          "enabled": false,
          "rootDir": "/sftp2/root",
          "authMethods": ["sftp_hostkey_2"],
          "protoConfig": {},
          "authorizedRules": {
            "sending": ["règle_envoi_1", "règle_envoi_2"],
            "reception": ["règle_récep_1", "règle_récep_2"]
          }
        },{
          "name": "sftp_server_1",
          "protocol": "sftp",
          "address": "localhost:2022",
          "enabled": true,
          "rootDir": "/sftp/root",
          "protoConfig": {},
          "authMethods": ["sftp_hostkey_1"],
          "authorizedRules": {
            "sending": ["règle_envoi_1", "règle_envoi_2"],
            "reception": ["règle_récep_1", "règle_récep_2"]
          }
        }]
      }
