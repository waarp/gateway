Lister les comptes partenaires
==============================

.. http:get:: /api/remote_accounts

   Renvoie une liste des comptes partenaires remplissant les critères donnés en
   paramètres de requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les comptes seront triés *(défaut: login+)*
   :type sort: [login+|login-]
   :param order: L'ordre dans lequel les comptes sont triés *(défaut: asc)*
   :type order: [asc|desc]
   :param agent: Filtre uniquement les comptes rattaché au partenaire portant ce numéro.
      Peut être renseigné plusieurs fois pour filtrer plusieurs partenaires.
   :type partner: uint64

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/remote_accounts?limit=10&order=desc&agent=1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array remoteAccounts: La liste des comptes demandés
   :resjsonarr number id: L'identifiant unique du compte
   :resjsonarr number remoteAgentID: L'identifiant unique du partenaire auquel
      le compte est rattaché
   :resjsonarr string Username: Le login du compte

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 147

          {
            "remoteAccounts": [{
              "id": 2,
              "remoteAgentID": 1,
              "login": "tutu",
            },{
              "id": 1,
              "remoteAgentID": 1,
              "login": "toto",
            }]
          }