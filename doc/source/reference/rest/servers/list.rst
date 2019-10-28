Lister les serveurs
======================

.. http:get:: /api/servers

   Renvoie une liste des serveurs remplissant les critères donnés en paramètres
   de requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sortby: Le paramètre selon lequel les serveurs seront triés *(défaut: name)*
   :type sortby: [name|protocol]
   :param order: L'ordre dans lequel les serveurs sont triés *(défaut: asc)*
   :type order: [asc|desc]
   :param protocol: Filtre uniquement les serveurs utilisant ce protocole.
      Peut être renseigné plusieurs fois pour filtrer plusieurs protocoles.
   :type protocol: [sftp]

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/servers?limit=10&order=desc&protocol=sftp HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array remoteAgents: La liste des serveurs demandés
   :resjsonarr number id: L'identifiant unique du serveur
   :resjsonarr string name: Le nom du serveur
   :resjsonarr [sftp] protocol: Le protocole utilisé par le serveur
   :resjsonarr string protoConfig: La configuration du serveur encodé dans une
      chaîne de caractères au format JSON.

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 228

          {
            "localAgents": [{
              "id": 2,
              "name": "sftp_server_2",
              "protocol": "sftp",
              "protoConfig": "{\"address\":\"localhost\",\"port\":22}
            },{
              "id": 1,
              "name": "sftp_server_1",
              "protocol": "sftp",
              "protoConfig": "{\"address\":\"localhost\",\"port\":21}
            }]
          }