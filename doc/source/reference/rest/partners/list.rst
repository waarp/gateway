Lister les partenaires
======================

.. http:get:: /api/partners

   Renvoie une liste des partenaires remplissant les critères donnés en paramètres
   de requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les partenaires seront triés *(défaut: name+)*
   :type sort: [name+|name-|protocol+|protocol-]
   :param protocol: Filtre uniquement les partenaires utilisant ce protocole.
      Peut être renseigné plusieurs fois pour filtrer plusieurs protocoles.
   :type protocol: [sftp]

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/partners?limit=10&order=desc&protocol=sftp HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array remoteAgents: La liste des partenaires demandés
   :resjsonarr number id: L'identifiant unique du partenaire
   :resjsonarr string name: Le nom du partenaire
   :resjsonarr [sftp] protocol: Le protocole utilisé par le partenaire
   :resjsonarr object protoConfig: La configuration du partenaire encodé sous
      forme d'un objet JSON.

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 267

          {
            "localAgents": [{
              "id": 2,
              "name": "waarp_sftp_2",
              "protocol": "sftp",
              "protoConfig": {
                "address": "waarp.fr",
                "port": 22,
                "root": "/sftp_2/root"
              }
            },{
              "id": 1,
              "name": "waarp_sftp",
              "protocol": "sftp",
              "protoConfig": {
                "address": "waarp.org",
                "port": 21,
                "root": "/sftp/root"
              }
            }]
          }