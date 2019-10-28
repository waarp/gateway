Lister les certificats
======================

.. http:get:: /api/certificates

   Renvoie une liste des certificats emplissant les critères donnés en paramètres
   de requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sortby: Le paramètre selon lequel les certificats seront triés *(défaut: name)*
   :type sortby: [name]
   :param order: L'ordre dans lequel les certificats sont triés *(défaut: asc)*
   :type order: [asc|desc]
   :param local_agents: Filtre uniquement les certificats rattaché au serveur
      local portant ce numéro. Peut être renseigné plusieurs fois pour filtrer
      plusieurs serveurs.
   :type local_agents: uint64
   :param remote_agents: Filtre uniquement les certificats rattaché au partenaire
      distant portant ce numéro. Peut être renseigné plusieurs fois pour filtrer
      plusieurs partenaires.
   :type remote_agents: uint64
   :param local_acounts: Filtre uniquement les certificats rattaché au compte
      local portant ce numéro. Peut être renseigné plusieurs fois pour filtrer
      plusieurs comptes.
   :type local_acounts: uint64
   :param remote_acounts: Filtre uniquement les certificats rattaché au compte
      partenaire portant ce numéro. Peut être renseigné plusieurs fois pour filtrer
      plusieurs comptes.
   :type remote_acounts: uint64

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/certificates?limit=5 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array certificates: La liste des certificats demandés
   :resjsonarr number id: L'identifiant unique du certificat
   :resjsonarr string name: Le nom du certificat
   :resjsonarr string ownerType: Le type d'entité
   :resjsonarr number ownerID: L'identifiant de l'entité à laquelle appartient le certificat
   :resjsonarr string privateKey: La clé privée de l'entité
   :resjsonarr string publicKey: La clé publique de l'entité
   :resjsonarr string cert: Le certificat de l'entité

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 453

          {
            "certificates": [{
              "id": 1,
              "name": "certificat_sftp_1",
              "ownerType": "local_agents",
              "ownerID": 1,
              "privateKey": "<clé privée 1>",
              "publicKey": "<clé publique 1>",
              "cert": "<certificat 1>"
            },{
              "id": 2,
              "name": "certificat_sftp_2",
              "ownerType": "local_agents",
              "ownerID": 1,
              "privateKey": "<clé privée 2>",
              "publicKey": "<clé publique 2>",
              "cert": "<certificat 2>"
            }]
          }