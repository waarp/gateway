Lister les comptes
==================

.. http:get:: /api/accounts

   Renvoie une liste des comptes remplissant les critères données en paramètres
   de requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sortby: Le paramètre selon lequel les comptes seront triés *(défaut: name)*
   :type sortby: [name]
   :param order: L'ordre dans lequel les compte sont triés *(défaut: asc)*
   :type order: [asc|desc]
   :param partner: Filtre uniquement les comptes rattaché au partenaire portant ce numéro.
                   Peut être renseigné plusieurs fois pour filtrer plusieurs partenaires.
   :type partner: uint64

   **Exemple de requête**

       .. code-block:: http

          GET /api/accounts?limit=10&order=desc HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array Accounts: La liste des comptes demandés
   :resjsonarr number ID: L'identifiant unique du compte
   :resjsonarr number PartnerID: L'identifiant unique du partenaire auquel le compte est rattaché
   :resjsonarr string Username: Le nom d'utilisateur du compte

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 147

          {
            "Accounts": [{
              "ID": 5678,
              "PartnerID": 67890,
              "Name": "partenaire2",
            },{
              "ID": 1234,
              "PartnerID": 12345,
              "Name": "partenaire1",
            }]
          }